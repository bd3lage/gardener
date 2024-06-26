// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strconv"
	"strings"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/features"
)

// ValidateName is a helper function for validating that a name is a DNS sub domain.
func ValidateName(name string, prefix bool) []string {
	return apivalidation.NameIsDNSSubdomain(name, prefix)
}

func validateSecretReference(ref corev1.SecretReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(ref.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must provide a name"))
	}
	if len(ref.Namespace) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("namespace"), "must provide a namespace"))
	}

	return allErrs
}

func validateCrossVersionObjectReference(ref autoscalingv1.CrossVersionObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(ref.APIVersion) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("apiVersion"), "must provide an apiVersion"))
	} else {
		if ref.APIVersion != corev1.SchemeGroupVersion.String() {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("apiVersion"), ref.APIVersion, []string{corev1.SchemeGroupVersion.String()}))
		}
	}

	if len(ref.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "must provide a kind"))
	} else {
		if ref.Kind != "Secret" && ref.Kind != "ConfigMap" {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("kind"), ref.Kind, []string{"Secret", "ConfigMap"}))
		}
	}

	if len(ref.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must provide a name"))
	}

	return allErrs
}

func validateNameConsecutiveHyphens(name string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if strings.Contains(name, "--") {
		allErrs = append(allErrs, field.Invalid(fldPath, name, "name may not contain two consecutive hyphens"))
	}

	return allErrs
}

// ValidateDNS1123Subdomain validates that a name is a proper DNS subdomain.
func ValidateDNS1123Subdomain(value string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range validation.IsDNS1123Subdomain(value) {
		allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
	}

	return allErrs
}

// validateDNS1123Label valides a name is a proper RFC1123 DNS label.
func validateDNS1123Label(value string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range validation.IsDNS1123Label(value) {
		allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
	}

	return allErrs
}

func getIntOrPercentValue(intOrStringValue intstr.IntOrString) int {
	value, isPercent := getPercentValue(intOrStringValue)
	if isPercent {
		return value
	}
	return intOrStringValue.IntValue()
}

func getPercentValue(intOrStringValue intstr.IntOrString) (int, bool) {
	if intOrStringValue.Type != intstr.String {
		return 0, false
	}
	if len(validation.IsValidPercent(intOrStringValue.StrVal)) != 0 {
		return 0, false
	}
	value, _ := strconv.Atoi(intOrStringValue.StrVal[:len(intOrStringValue.StrVal)-1])
	return value, true
}

var availableFailureTolerance = sets.New(
	string(core.FailureToleranceTypeNode),
	string(core.FailureToleranceTypeZone),
)

// ValidateFailureToleranceTypeValue validates if the passed value is a valid failureToleranceType.
func ValidateFailureToleranceTypeValue(value core.FailureToleranceType, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	failureToleranceType := string(value)
	if !availableFailureTolerance.Has(failureToleranceType) {
		allErrs = append(allErrs, field.NotSupported(fldPath, failureToleranceType, sets.List(availableFailureTolerance)))
	}

	return allErrs
}

var availableIPFamilies = sets.New(
	string(core.IPFamilyIPv4),
	string(core.IPFamilyIPv6),
)

// ValidateIPFamilies validates the given list of IP families for valid values and combinations.
func ValidateIPFamilies(ipFamilies []core.IPFamily, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	ipFamiliesSeen := sets.New[string]()
	for i, ipFamily := range ipFamilies {
		// validate: only supported IP families
		if !availableIPFamilies.Has(string(ipFamily)) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Index(i), ipFamily, sets.List(availableIPFamilies)))
		}

		// validate: no duplicate IP families
		if ipFamiliesSeen.Has(string(ipFamily)) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(i), ipFamily))
		} else {
			ipFamiliesSeen.Insert(string(ipFamily))
		}
	}

	if len(allErrs) > 0 {
		// further validation doesn't make any sense, because there are unsupported or duplicate IP families
		return allErrs
	}

	if len(ipFamilies) > 0 && ipFamilies[0] == core.IPFamilyIPv6 && !features.DefaultFeatureGate.Enabled(features.IPv6SingleStack) {
		allErrs = append(allErrs, field.Invalid(fldPath, ipFamilies, "IPv6 single-stack networking is not supported"))
	}

	return allErrs
}
