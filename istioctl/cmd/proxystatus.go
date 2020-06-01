// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"istio.io/istio/istioctl/pkg/clioptions"
	"istio.io/istio/istioctl/pkg/kubernetes"
	"istio.io/istio/istioctl/pkg/util/handlers"
	"istio.io/istio/istioctl/pkg/writer/compare"
	"istio.io/istio/istioctl/pkg/writer/pilot"
)

func statusCommand() *cobra.Command {
	var opts clioptions.ControlPlaneOptions

	statusCmd := &cobra.Command{
		Use:   "proxy-status [<pod-name[.namespace]>]",
		Short: "Retrieves the synchronization status of each Envoy in the mesh [kube only]",
		Long: `
Retrieves last sent and last acknowledged xDS sync from Pilot to each Envoy in the mesh

`,
		Example: `# Retrieve sync status for all Envoys in a mesh
	istioctl proxy-status

# Retrieve sync diff for a single Envoy and Pilot
	istioctl proxy-status istio-egressgateway-59585c5b9c-ndc59.istio-system
`,
		Aliases: []string{"ps"},
		RunE: func(c *cobra.Command, args []string) error {
			kubeClient, err := clientExecFactory(kubeconfig, configContext, opts)
			if err != nil {
				return err
			}
			if len(args) > 0 {
				podName, ns := handlers.InferPodInfo(args[0], handlers.HandleNamespace(namespace, defaultNamespace))
				path := "config_dump"
				envoyDump, err := kubeClient.EnvoyDo(podName, ns, "GET", path, nil)
				if err != nil {
					return err
				}

				path = fmt.Sprintf("/debug/config_dump?proxyID=%s.%s", podName, ns)
				pilotDumps, err := kubeClient.AllPilotsDiscoveryDo(istioNamespace, path)
				if err != nil {
					return err
				}
				c, err := compare.NewComparator(c.OutOrStdout(), pilotDumps, envoyDump)
				if err != nil {
					return err
				}
				return c.Diff()
			}
			statuses, err := kubeClient.AllPilotsDiscoveryDo(istioNamespace, "/debug/syncz")
			if err != nil {
				return err
			}
			sw := pilot.StatusWriter{Writer: c.OutOrStdout()}
			return sw.PrintAll(statuses)
		},
	}

	opts.AttachControlPlaneFlags(statusCmd)

	return statusCmd
}

func newPilotExecClient(kubeconfig, configContext string, opts clioptions.ControlPlaneOptions) (kubernetes.ExecClient, error) {
	return kubernetes.NewExtendedClient(kubeconfig, configContext, opts)
}

func newEnvoyClient(kubeconfig, configContext string) (kubernetes.ExecClient, error) {
	return kubernetes.NewClient(kubeconfig, configContext)
}
