package plugin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

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
)

const (
	Deny     = "-"
	Wildcard = "*"
	Ingress  = "Ingress"
	Egress   = "Egress"
)

type SourceType int

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

const (
	PodSelector             SourceType = 1
	NamespaceSelector       SourceType = 2
	IpBlock                 SourceType = 3
	PodAndNameSpaceSelector SourceType = 4
)

// Runs the plugin
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

	isIngress := getFlagBool(cmd, "ingress")
	isEgress := getFlagBool(cmd, "egress")
	podName := util.GetFlagString(cmd, "pod")

	if getFlagBool(cmd, "all-namespaces") {
		namespace = ""
	}

	networkPolicies, err := getNetworkPolicies(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to list network policies")
	}

	var tableLines []TableLine
	for _, policy := range networkPolicies.Items {

		if isIngress || (!isIngress && !isEgress) {
			// If Default Deny
			if containsPolicyTypes(policy.Spec.PolicyTypes, netv1.PolicyTypeIngress) && policy.Spec.Ingress == nil {
				tableLines = append(tableLines, createTableLineWithDeny(policy, Ingress))
			}

			for _, ingresses := range policy.Spec.Ingress {
				// If Wide Open
				if ingresses.From == nil && ingresses.Ports == nil {
					tableLines = append(tableLines, createTableLineWithWildcard(policy, Ingress))
					continue
				}

				for _, peer := range ingresses.From {
					if peer.PodSelector != nil && peer.NamespaceSelector != nil {
						tableLines = append(tableLines, createTableLineForSourceType(policy, peer, ingresses.Ports,
							Ingress, PodAndNameSpaceSelector))
					} else {
						if peer.PodSelector != nil {
							tableLines = append(tableLines, createTableLineForSourceType(policy, peer, ingresses.Ports,
								Ingress, PodSelector))
						}
						if peer.NamespaceSelector != nil {
							tableLines = append(tableLines, createTableLineForSourceType(policy, peer, ingresses.Ports,
								Ingress, NamespaceSelector))
						}
					}
					if peer.IPBlock != nil {
						tableLines = append(tableLines, createTableLineForSourceType(policy, peer, ingresses.Ports,
							Ingress, IpBlock))
					}
				}
				if len(ingresses.Ports) > 0 && len(ingresses.From) == 0 {
					tableLines = append(tableLines, createTableLineForPortBlock(policy, ingresses.Ports, Ingress))
				}
			}
		}

		if isEgress || (!isEgress && !isIngress) {
			// If Default Deny
			if containsPolicyTypes(policy.Spec.PolicyTypes, netv1.PolicyTypeEgress) && policy.Spec.Egress == nil {
				tableLines = append(tableLines, createTableLineWithDeny(policy, Egress))
			}

			for _, egresses := range policy.Spec.Egress {
				// If Wide Open
				if egresses.To == nil && egresses.Ports == nil {
					tableLines = append(tableLines, createTableLineWithWildcard(policy, Egress))
					continue
				}

				for _, peer := range egresses.To {
					if peer.PodSelector != nil && peer.NamespaceSelector != nil {
						tableLines = append(tableLines, createTableLineForSourceType(policy, peer, egresses.Ports,
							Egress, PodAndNameSpaceSelector))
					} else {
						if peer.PodSelector != nil {
							tableLines = append(tableLines, createTableLineForSourceType(policy, peer, egresses.Ports,
								Egress, PodSelector))
						}
						if peer.NamespaceSelector != nil {
							tableLines = append(tableLines, createTableLineForSourceType(policy, peer, egresses.Ports,
								Egress, NamespaceSelector))
						}
					}
					if peer.IPBlock != nil {
						tableLines = append(tableLines, createTableLineForSourceType(policy, peer, egresses.Ports,
							Egress, IpBlock))
					}
				}
				if len(egresses.Ports) > 0 && len(egresses.To) == 0 {
					tableLines = append(tableLines, createTableLineForPortBlock(policy, egresses.Ports, Egress))
				}
			}
		}
	}

	if len(podName) > 0 {
		pod, err := getPod(clientset, namespace, podName)
		if err != nil {
			return errors.Wrap(err, "failed getting pod")
		}
		tableLines = filterLinesBasedOnPodLabels(tableLines, pod)
	}

	if len(tableLines) == 0 {
		return errors.New("No network policy was found")
	}

	renderTable(tableLines)
	return nil
}

// Creates a new line for the result table
func createTableLine(policy netv1.NetworkPolicy, ports []netv1.NetworkPolicyPort,
	policyType string) TableLine {

	var line TableLine
	line.networkPolicyName = policy.Name
	line.namespace = policy.Namespace
	line.policyType = policyType

	if policy.Spec.PodSelector.Size() == 0 {
		line.pods = Wildcard
	} else {
		line.pods = sortAndJoinLabels(policy.Spec.PodSelector)
	}

	if len(ports) == 0 {
		line.policyPort = Wildcard
	} else {
		for _, port := range ports {
			line.policyPort = addCharIfNotEmpty(line.policyPort, "\n") +
				fmt.Sprintf("%s:%s", getProtocol(*port.Protocol), port.Port)
		}
	}
	return line
}

func createTableLineWithDeny(policy netv1.NetworkPolicy, policyType string) TableLine {
	line := createTableLine(policy, []netv1.NetworkPolicyPort{}, policyType)
	line.policyPods = Deny
	line.policyIpBlock = Deny
	line.policyNamespace = Deny
	line.policyPort = Deny
	return line
}

func createTableLineWithWildcard(policy netv1.NetworkPolicy, policyType string) TableLine {
	line := createTableLine(policy, []netv1.NetworkPolicyPort{}, policyType)
	line.policyPods = Wildcard
	line.policyIpBlock = Wildcard
	line.policyNamespace = Wildcard
	return line
}

// Creates a new line for the result table for a specific source type
func createTableLineForSourceType(policy netv1.NetworkPolicy, peer netv1.NetworkPolicyPeer, ports []netv1.NetworkPolicyPort,
	policyType string, sourceType SourceType) TableLine {

	line := createTableLine(policy, ports, policyType)

	if sourceType == PodSelector {
		line.policyPods = sortAndJoinLabels(*peer.PodSelector)
		line.policyNamespace = line.namespace
		line.policyIpBlock = Wildcard
	}

	if sourceType == NamespaceSelector {
		line.policyNamespace = sortAndJoinLabels(*peer.NamespaceSelector)
		line.policyPods = Wildcard
		line.policyIpBlock = Wildcard
	}

	if sourceType == PodAndNameSpaceSelector {
		line.policyNamespace = sortAndJoinLabels(*peer.NamespaceSelector)
		line.policyPods = sortAndJoinLabels(*peer.PodSelector)
		line.policyIpBlock = Wildcard
	}

	if sourceType == IpBlock {
		var exceptions string
		for _, exception := range peer.IPBlock.Except {
			exceptions = addCharIfNotEmpty(exceptions, "\n") + exception
		}
		line.policyIpBlock = fmt.Sprintf("CIDR: %s Except: [%s]", peer.IPBlock.CIDR, exceptions)
		line.policyPods = Wildcard
		line.policyNamespace = Wildcard
	}

	return line
}

// Creates a new line for the result table for a rule that only have ports
func createTableLineForPortBlock(policy netv1.NetworkPolicy, ports []netv1.NetworkPolicyPort,
	policyType string) TableLine {

	line := createTableLine(policy, ports, policyType)
	line.policyNamespace = Wildcard
	line.policyPods = Wildcard
	line.policyIpBlock = Wildcard
	return line
}

// Sorts and joins the labels with a new space delimiter based on podSelector field
func sortAndJoinLabels(podSelector metav1.LabelSelector) string {

	if len(podSelector.MatchExpressions) != 0 {
		return sortAndJoinLabelsForMatchExpressions(podSelector.MatchExpressions)
	}

	return sortAndJoinLabelsForMatchLabels(podSelector.MatchLabels)
}

// Sorts and joins the labels with a new space delimiter by parsing MatchLabels field
func sortAndJoinLabelsForMatchLabels(labels map[string]string) string {
	result := ""
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		result = addCharIfNotEmpty(result, "\n") + fmt.Sprintf("%s=%s", k, labels[k])
	}

	return result
}

// Sorts and joins the labels with a new space delimiter by parsing MatchExpressions field
// possible operators: Exists, DoesNotExist, In, NotIn
func sortAndJoinLabelsForMatchExpressions(matchExpressions []metav1.LabelSelectorRequirement) string {
	result := ""
	for _, expression := range matchExpressions {
		key := expression.Key
		switch expression.Operator {
		case metav1.LabelSelectorOpExists:
			result = addCharIfNotEmpty(result, "\n") + fmt.Sprintf("%s=%s", key, "*")
		case metav1.LabelSelectorOpDoesNotExist:
			result = addCharIfNotEmpty(result, "\n") + fmt.Sprintf("^(%s)=%s", key, "*")
		case metav1.LabelSelectorOpIn:
			labelValues := "(" + strings.Join(expression.Values, "|") + ")"
			result = addCharIfNotEmpty(result, "\n") + fmt.Sprintf("%s=%s", key, labelValues)
		case metav1.LabelSelectorOpNotIn:
			labelValues := "(" + strings.Join(expression.Values, "|") + ")"
			result = addCharIfNotEmpty(result, "\n") + fmt.Sprintf("%s=%s", key, "^"+labelValues)
		}
	}

	return result
}

// Returns the list of network policies
func getNetworkPolicies(clientset *kubernetes.Clientset, namespace string) (result *netv1.NetworkPolicyList,
	err error) {

	return clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(),
		metav1.ListOptions{})
}

// Returns the pod based on the name and namespace
func getPod(clientset *kubernetes.Clientset, namespace string, podName string) (result *corev1.Pod, err error) {
	selector := fields.OneTermEqualSelector("metadata.name", podName)
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(),
		metav1.ListOptions{FieldSelector: selector.String()})

	if len(podList.Items) == 0 {
		err = errors.New(fmt.Sprintf("Pods \"%s\" not found", podName))
	} else {
		result = &podList.Items[0]
	}
	return
}

// Renders the result table
func renderTable(tableLines []TableLine) {
	var data [][]string
	for _, line := range tableLines {
		stringLine := []string{line.networkPolicyName, line.policyType, line.namespace, line.pods, line.policyNamespace,
			line.policyPods, line.policyIpBlock, line.policyPort}
		data = append(data, stringLine)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Network Policy", "Type", "Namespace", "Pods", "Namespaces Selector", "Pods Selector",
		"IP Block", "Ports"})
	table.SetAutoMergeCells(false)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.AppendBulk(data)
	table.Render()
}

// Returns the protocol as string
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

// Adds the char c to the string s if the string s is not empty
func addCharIfNotEmpty(s string, c string) string {
	if len(s) > 0 {
		return s + c
	}
	return s
}

// Gets the the flag value as a boolean, otherwise returns false if the flag value is nil
func getFlagBool(cmd *cobra.Command, flag string) bool {
	b, err := cmd.Flags().GetBool(flag)
	if err != nil {
		return false
	}
	return b
}

// Filters lines in the result table based on the pod labels
func filterLinesBasedOnPodLabels(tableLines []TableLine, pod *corev1.Pod) []TableLine {
	var filteredTable []TableLine
	for _, line := range tableLines {
		if line.pods != Wildcard {
			labels := strings.Split(line.pods, "\n")
			appendLine := true
			for _, labelCondition := range labels {
				if !checkLabelCondition(labelCondition, pod) {
					appendLine = false
					break
				}
			}
			if appendLine {
				filteredTable = append(filteredTable, line)
			}
		} else {
			filteredTable = append(filteredTable, line)
		}
	}
	return filteredTable
}

// checkLabelCondition: check that a single label selector condition line is satisfied given a pod spec.
// It support matchLabels and matchExpressions conditions type
func checkLabelCondition(labelCondition string, pod *corev1.Pod) bool {
	keyValue := strings.Split(labelCondition, "=")
	key := keyValue[0]
	value := keyValue[1]
	if strings.HasPrefix(key, "^(") { // Label line: '^(label)=*'
		return checkDoesNotExistCondition(key, pod)
	} else if value == "*" { // prefix should be != '^(' also, Label line: 'label=*'
		return checkExistCondition(key, pod)
	} else if strings.HasPrefix(value, "^(") { // Label line: 'label=(value1|...|valueN)'
		return checkNotInCondition(key, value, pod)
	} else if strings.HasPrefix(value, "(") { // Label line: 'label=^(value1|...|valueN)'
		return checkInCondition(key, value, pod)
	} else if pod.Labels[keyValue[0]] != keyValue[1] { // simple label filter
		return false
	}

	return true
}

// checkExistCondition: check an Exist filter against a pod spec. label line: 'label=*'.
// Return true if the label key exist in pod spec
func checkExistCondition(key string, pod *corev1.Pod) bool {
	key = strings.TrimSuffix(strings.TrimPrefix(key, "("), ")")
	_, exist := pod.Labels[key]
	return exist
}

// checkDoesNotExistCondition: check a DoesNotExist filter against a pod spec. Label line: '^(label)=*'
// // Return true if the label key does not exist in pod spec
func checkDoesNotExistCondition(key string, pod *corev1.Pod) bool {
	isolateKey := strings.TrimSuffix(strings.TrimPrefix(key, "^("), ")")
	return !checkExistCondition(isolateKey, pod)
}

// checkInCondition: check an NotIn filter against a pod spec. label line: 'label=(value1|...|valueN)'
// Return true if the label key if and only if the label exist and does not have specific values
func checkInCondition(key, value string, pod *corev1.Pod) bool {
	podLabelValue, exist := pod.Labels[key]
	if !exist {
		return false
	}

	values := strings.Split(strings.TrimSuffix(strings.TrimPrefix(value, "("), ")"), "|")
	for _, value := range values {
		if value == podLabelValue {
			return true
		}
	}
	return false
}

// checkNotInCondition: check an NotIn filter against a pod spec. label line: 'label=^(value1|...|valueN)'
// Return true if the label key is not set in pod OR do not have specific values
func checkNotInCondition(key, value string, pod *corev1.Pod) bool {
	return !checkInCondition(key, strings.TrimPrefix(value, "^"), pod)
}

// Returns true if the slice contains the policy type
func containsPolicyTypes(s []netv1.PolicyType, value netv1.PolicyType) bool {
	for _, a := range s {
		if a == value {
			return true
		}
	}
	return false
}
