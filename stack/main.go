package stack

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/adamgoose/tsk/config"
	k8sappsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	k8scorev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	k8smetav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	k8srbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi-tailscale/sdk/go/tailscale"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	. "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed Corefile
var Corefile string

func GetStack(ctx context.Context, cfg config.Config) (auto.Stack, error) {
	project := auto.Project(workspace.Project{
		Name:    tokens.PackageName("tailscale-k8s"),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		Backend: &workspace.ProjectBackend{
			URL: cfg.StorageDir,
		},
	})

	envVars := auto.EnvVars(map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "tailscale-k8s",
	})

	// consider using context as stack name
	s, err := auto.UpsertStackInlineSource(ctx, cfg.Tailscale.Hostname, "tailscale-k8s", TailscaleRunFunc(cfg), project, envVars)
	if err != nil {
		return s, err
	}

	if err := s.Workspace().InstallPlugin(ctx, "kubernetes", "v3.8.2"); err != nil {
		return s, err
	}

	if err := s.Workspace().InstallPlugin(ctx, "tailscale", "v0.1.0"); err != nil {
		return s, err
	}

	s.SetConfig(ctx, "tailscale:apiKey", auto.ConfigValue{
		Value:  cfg.Tailscale.APIKey,
		Secret: true,
	})
	s.SetConfig(ctx, "tailscale:tailnet", auto.ConfigValue{
		Value:  cfg.Tailscale.Tailnet,
		Secret: false,
	})

	return s, nil
}

func TailscaleRunFunc(config config.Config) RunFunc {
	return func(ctx *Context) error {
		name := Sprintf("tsk-%s", config.Kubernetes.Username)

		sa, err := k8scorev1.NewServiceAccount(ctx, "service-account", &k8scorev1.ServiceAccountArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      name,
				Namespace: String(config.Kubernetes.Namespace),
			},
		})
		if err != nil {
			return err
		}

		role, err := k8srbacv1.NewRole(ctx, "role", &k8srbacv1.RoleArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      name,
				Namespace: String(config.Kubernetes.Namespace),
			},
			Rules: k8srbacv1.PolicyRuleArray{k8srbacv1.PolicyRuleArgs{
				ApiGroups:     ToStringArray([]string{""}),
				Resources:     ToStringArray([]string{"secrets"}),
				ResourceNames: StringArray{name},
				Verbs:         ToStringArray([]string{"get", "create", "update", "patch"}),
			}, k8srbacv1.PolicyRuleArgs{
				ApiGroups:     ToStringArray([]string{"apps"}),
				Resources:     ToStringArray([]string{"deployments"}),
				ResourceNames: StringArray{name},
				Verbs:         ToStringArray([]string{"get", "update", "patch"}),
			}},
		}, Parent(sa))
		if err != nil {
			return err
		}

		k8srbacv1.NewRoleBinding(ctx, "role-binding", &k8srbacv1.RoleBindingArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      role.Metadata.Name().Elem(),
				Namespace: sa.Metadata.Namespace().Elem(),
			},
			Subjects: k8srbacv1.SubjectArray{k8srbacv1.SubjectArgs{
				Kind:      String("ServiceAccount"),
				Name:      sa.Metadata.Name().Elem(),
				Namespace: sa.Metadata.Namespace().Elem(),
			}},
			RoleRef: k8srbacv1.RoleRefArgs{
				ApiGroup: String("rbac.authorization.k8s.io"),
				Kind:     String("Role"),
				Name:     role.Metadata.Name().Elem(),
			},
		}, Parent(sa))

		sec, err := k8scorev1.NewSecret(ctx, "secret", &k8scorev1.SecretArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      name,
				Namespace: sa.Metadata.Namespace().Elem(),
			},
			StringData: ToStringMap(map[string]string{
				"TSK_EPHEMERAL_KEY": config.Tailscale.EphemeralKey,
			}),
		}, Parent(sa), DeleteBeforeReplace(true))
		if err != nil {
			return err
		}

		tpl := template.Must(template.New("Corefile").Parse(Corefile))
		cf := bytes.NewBuffer(nil)
		if err := tpl.Execute(cf, config.Kubernetes); err != nil {
			return err
		}
		cm, err := k8scorev1.NewConfigMap(ctx, "configmap", &k8scorev1.ConfigMapArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      name,
				Namespace: sa.Metadata.Namespace().Elem(),
			},
			Data: ToStringMap(map[string]string{
				"Corefile": cf.String(),
			}),
		}, Parent(sa))
		if err != nil {
			return err
		}

		labels := ToStringMap(map[string]string{
			"app":     "tsk",
			"user":    config.Kubernetes.Username,
			"tailnet": config.Tailscale.Tailnet,
		})

		dp, err := k8sappsv1.NewDeployment(ctx, "deployment", &k8sappsv1.DeploymentArgs{
			Metadata: &k8smetav1.ObjectMetaArgs{
				Name:      name,
				Namespace: sa.Metadata.Namespace().Elem(),
				Labels:    labels,
			},
			Spec: &k8sappsv1.DeploymentSpecArgs{
				Replicas: Int(1),
				Strategy: k8sappsv1.DeploymentStrategyArgs{
					RollingUpdate: k8sappsv1.RollingUpdateDeploymentArgs{
						MaxSurge:       String("0%"),
						MaxUnavailable: String("100%"),
					},
				},
				Selector: &k8smetav1.LabelSelectorArgs{MatchLabels: labels},
				Template: &k8scorev1.PodTemplateSpecArgs{
					Metadata: &k8smetav1.ObjectMetaArgs{Labels: labels},
					Spec: &k8scorev1.PodSpecArgs{
						ServiceAccountName: sa.Metadata.Name().Elem(),
						Volumes: k8scorev1.VolumeArray{k8scorev1.VolumeArgs{
							Name: String("corefile"),
							ConfigMap: k8scorev1.ConfigMapVolumeSourceArgs{
								Name: cm.Metadata.Name().Elem(),
							},
						}},
						Containers: k8scorev1.ContainerArray{
							k8scorev1.ContainerArgs{
								Name:            String("coredns"),
								Image:           String("coredns/coredns"),
								ImagePullPolicy: String("Always"),
								Args:            ToStringArray([]string{"-conf", "/etc/coredns/Corefile"}),
								VolumeMounts: k8scorev1.VolumeMountArray{k8scorev1.VolumeMountArgs{
									Name:      String("corefile"),
									MountPath: String("/etc/coredns"),
									ReadOnly:  Bool(true),
								}},
								Ports: k8scorev1.ContainerPortArray{
									k8scorev1.ContainerPortArgs{
										Name:          String("liveness"),
										ContainerPort: Int(8080),
									},
									k8scorev1.ContainerPortArgs{
										Name:          String("readiness"),
										ContainerPort: Int(8181),
									},
								},
								LivenessProbe: k8scorev1.ProbeArgs{
									HttpGet: k8scorev1.HTTPGetActionArgs{
										Path:   String("/health"),
										Port:   Int(8080),
										Scheme: String("HTTP"),
									},
								},
								ReadinessProbe: k8scorev1.ProbeArgs{
									HttpGet: k8scorev1.HTTPGetActionArgs{
										Path:   String("/ready"),
										Port:   Int(8181),
										Scheme: String("HTTP"),
									},
								},
							},
							k8scorev1.ContainerArgs{
								Name:            String("tailscaled"),
								Image:           String("tailscale/tailscale"),
								ImagePullPolicy: String("Always"),
								Env: k8scorev1.EnvVarArray{
									k8scorev1.EnvVarArgs{
										Name:  String("TSK_KUBE_SECRET"),
										Value: sec.Metadata.Name().Elem(),
									},
									k8scorev1.EnvVarArgs{
										Name:  String("TSK_CIDR"),
										Value: String(config.Kubernetes.ServiceCIDR),
									}, k8scorev1.EnvVarArgs{
										Name: String("TSK_EPHEMERAL_KEY"),
										ValueFrom: k8scorev1.EnvVarSourceArgs{
											SecretKeyRef: k8scorev1.SecretKeySelectorArgs{
												Name: sec.Metadata.Name().Elem(),
												Key:  String("TSK_EPHEMERAL_KEY"),
											},
										},
									}, k8scorev1.EnvVarArgs{
										Name:  String("TSK_HOSTNAME"),
										Value: String(config.Tailscale.Hostname),
									},
								},
								SecurityContext: k8scorev1.SecurityContextArgs{
									RunAsUser:  Int(1000),
									RunAsGroup: Int(1000),
								},
								Command: ToStringArray([]string{
									"sh", "-c", "tailscaled --state=kube:$TSK_KUBE_SECRET --socket=/tmp/tailscaled.sock --tun=userspace-networking",
								}),
								Lifecycle: k8scorev1.LifecycleArgs{
									PostStart: k8scorev1.HandlerArgs{
										Exec: k8scorev1.ExecActionArgs{
											Command: ToStringArray([]string{
												"sh", "-c", "tailscale --socket=/tmp/tailscaled.sock up --accept-dns=false --advertise-routes=$TSK_CIDR --authkey=$TSK_EPHEMERAL_KEY --hostname=$TSK_HOSTNAME",
											}),
										},
									},
								},
							},
						},
					},
				},
			},
		}, Parent(sa))
		if err != nil {
			return err
		}

		deviceList, err := tailscale.GetDevices(ctx, &tailscale.GetDevicesArgs{
			NamePrefix: &config.Tailscale.Hostname,
		})
		if err != nil {
			return fmt.Errorf("Couldn't fetch devices: %v", err)
		}

		dnsConfigured := false
		for _, device := range deviceList.Devices {
			if device.Name != fmt.Sprintf("%s.%s", config.Tailscale.Hostname, config.Tailscale.Tailnet) {
				continue
			}

			subnet, err := tailscale.NewDeviceSubnetRoutes(ctx, device.Name, &tailscale.DeviceSubnetRoutesArgs{
				DeviceId: String(device.Id),
				Routes:   ToStringArray([]string{config.Kubernetes.ServiceCIDR}),
			}, Parent(dp))
			if err != nil {
				return fmt.Errorf("Couldn't create device: %v", err)
			}

			ns, err := tailscale.NewDnsNameservers(ctx, device.Name, &tailscale.DnsNameserversArgs{
				Nameservers: StringArray{String(device.Addresses[0])},
			}, Parent(subnet))
			if err != nil {
				return fmt.Errorf("Couldn't register DNS nameservers: %v", err)
			}

			_, err = tailscale.NewDnsSearchPaths(ctx, device.Name, &tailscale.DnsSearchPathsArgs{
				SearchPaths: StringArray{String("tsk")},
			}, Parent(ns), DependsOn([]Resource{ns}))
			if err != nil {
				return fmt.Errorf("Couldn't register DNS search paths: %v", err)
			}

			dnsConfigured = true

			break
		}

		ctx.Export("dnsConfigured", Bool(dnsConfigured))
		return nil
	}
}
