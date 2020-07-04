package plugin

import (
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
	"os"
	"strings"
)

const (
	ALL_VALUES = "*"
)

type TableLine struct {
	networkPolicyName string
	namespace         string
	pods              string
	policyType        string
	policyNamespace   string
	policyPods        string
	policyIpBlock     string
	policyPort        string
}

func RunPlugin(configFlags *genericclioptions.ConfigFlags, cmd *cobra.Command) error {
	factory := util.NewFactory(configFlags)
	clientConfig := factory.ToRawKubeConfigLoader()
	config, err := factory.ToRESTConfig()

	if err != nil {
		return errors.Wrap(err, "failed to read kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return errors.WithMessage(err, "Failed getting namespace")
	}

	ingress := getFlagBool(cmd, "ingress")
	egress := getFlagBool(cmd, "egress")
	allNamespace := getFlagBool(cmd, "all-namespaces")
	podName := util.GetFlagString(cmd, "pod")

	if allNamespace {
		namespace = ""
	}

	networkPolicies, err := getNetworkPolicies(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to list network policies")
	}

	var tableLines []TableLine
	for _, policy := range networkPolicies.Items {

		if ingress || (!ingress && !egress) {
			for _, ingresses := range policy.Spec.Ingress {
				for _, peer := range ingresses.From {
					if peer.PodSelector != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, ingresses.Ports, "ingress", "PodSelector"))
					}
					if peer.NamespaceSelector != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, ingresses.Ports, "ingress", "NamespaceSelector"))
					}
					if peer.IPBlock != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, ingresses.Ports, "ingress", "IPBlock"))
					}
				}
			}
		}

		if egress || (!egress && !ingress) {
			for _, egresses := range policy.Spec.Egress {
				for _, peer := range egresses.To {
					if peer.PodSelector != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, egresses.Ports, "egress", "PodSelector"))
					}
					if peer.NamespaceSelector != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, egresses.Ports, "egress", "NamespaceSelector"))
					}
					if peer.IPBlock != nil {
						tableLines = append(tableLines, createTableLine(policy, peer, egresses.Ports, "egress", "IPBlock"))
					}
				}
			}
		}
	}

	if len(podName) > 0 {
		pod, err := getPod(clientset, namespace, podName)
		if err != nil {
			return errors.Wrap(err, "failed getting pod")
		}
		tableLines = filterToPod(tableLines, pod)
	}

	renderTable(tableLines)
	return nil
}

func createTableLine(policy netv1.NetworkPolicy, peer netv1.NetworkPolicyPeer, ports []netv1.NetworkPolicyPort, policyType string, sourceType string) TableLine {
	var line TableLine
	line.networkPolicyName = policy.Name
	line.namespace = policy.Namespace
	line.policyType = policyType

	if policy.Spec.PodSelector.Size() == 0 {
		line.pods = ALL_VALUES
	} else {
		for k, v := range policy.Spec.PodSelector.MatchLabels {
			line.pods = addCharIfNotEmpty(line.pods, "\n") + fmt.Sprintf("%s=%s", k, v)
		}
	}

	if len(ports) == 0 {
		line.policyPort = ALL_VALUES
	} else {
		for _, port := range ports {
			line.policyPort = addCharIfNotEmpty(line.policyPort, "\n") + fmt.Sprintf("%s:%s", getProtocol(*port.Protocol), port.Port)
		}
	}

	if sourceType == "PodSelector" {
		for k, v := range peer.PodSelector.MatchLabels {
			line.policyPods = addCharIfNotEmpty(line.policyPods, "\n") + fmt.Sprintf("%s=%s", k, v)
		}
		line.policyNamespace = line.namespace
		line.policyIpBlock = ALL_VALUES
	}

	if sourceType == "NamespaceSelector" {
		for k, v := range peer.NamespaceSelector.MatchLabels {
			line.policyNamespace = addCharIfNotEmpty(line.policyNamespace, "\n") + fmt.Sprintf("%s=%s", k, v)
		}
		line.policyPods = ALL_VALUES
		line.policyIpBlock = ALL_VALUES
	}

	if sourceType == "IPBlock" {
		var exceptions string
		for _, exception := range peer.IPBlock.Except {
			exceptions = addCharIfNotEmpty(exceptions, "\n") + exception
		}
		line.policyIpBlock = fmt.Sprintf("CIDR: %s Except: [%s]", peer.IPBlock.CIDR, exceptions)
		line.policyPods = ALL_VALUES
		line.policyNamespace = ALL_VALUES
	}

	return line
}

func getNetworkPolicies(clientset *kubernetes.Clientset, namespace string) (result *netv1.NetworkPolicyList, err error) {
	return clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(),
		metav1.ListOptions{})
}

func getPod(clientset *kubernetes.Clientset, namespace string, podName string) (result *corev1.Pod, err error) {
	selector := fields.OneTermEqualSelector("metadata.name", podName)
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: selector.String()})

	if len(podList.Items) == 0 {
		err = errors.New("Failed getting pod")
	} else {
		result = &podList.Items[0]
	}
	return
}

func renderTable(tableLines []TableLine) {
	var data [][]string
	for _, line := range tableLines {
		stringLine := []string{line.networkPolicyName, line.policyType, line.namespace, line.pods, line.policyNamespace,
			line.policyPods, line.policyIpBlock, line.policyPort}
		data = append(data, stringLine)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Network Policy", "Type", "Namespace", "Pods", "Namespaces Selector", "Pods Selector", "IP Block", "Ports"})
	table.SetAutoMergeCells(false)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.AppendBulk(data)
	table.Render()
}

func getProtocol(protocol corev1.Protocol) string {
	switch protocol {
	case corev1.ProtocolSCTP:
		return "SCTP"
	case corev1.ProtocolUDP:
		return "UDP"
	case corev1.ProtocolTCP:
		return "TCP"
	default:
		return ""
	}
}

func addCharIfNotEmpty(s string, char string) string {
	if len(s) > 0 {
		return s + char
	}
	return s
}

func getFlagBool(cmd *cobra.Command, flag string) bool {
	b, err := cmd.Flags().GetBool(flag)
	if err != nil {
		return false
	}
	return b
}

func filterToPod(tableLines []TableLine, pod *corev1.Pod) []TableLine {
	var filteredTable []TableLine
	for _, line := range tableLines {
		if line.pods != ALL_VALUES {
			labels := strings.Split(line.pods, "\n")
			for _, label := range labels {
				keyValue := strings.Split(label, "=")
				if pod.Labels[keyValue[0]] == keyValue[1] {
					filteredTable = append(filteredTable, line)
				}
			}
		} else {
			filteredTable = append(filteredTable, line)
		}
	}
	return filteredTable
}
