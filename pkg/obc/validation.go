package obc

import (
	"encoding/json"
	"fmt"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/nb"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	"github.com/noobaa/noobaa-operator/v5/pkg/validations"
)

// ValidateOBC validate object bucket claim
func ValidateOBC(obc *nbv1.ObjectBucketClaim, isCLI bool) error {
	if obc == nil {
		return nil
	}
	return validateAdditionalConfig(obc.Name, obc.Spec.AdditionalConfig, false, isCLI)
}

// ValidateOB validate object bucket
func ValidateOB(ob *nbv1.ObjectBucket, isCLI bool) error {
	if ob == nil {
		return nil
	}
	return validateAdditionalConfig(ob.Name, ob.Spec.Endpoint.AdditionalConfigData, true, isCLI)
}

// Validate additional config
func validateAdditionalConfig(objectName string, additionalConfig map[string]string, update bool, isCLI bool) error {
	if additionalConfig == nil {
		return nil
	}

	obcMaxSize := additionalConfig["maxSize"]
	obcMaxObjects := additionalConfig["maxObjects"]
	replicationPolicy := additionalConfig["replicationPolicy"]
	NSFSAccountConfig := additionalConfig["nsfsAccountConfig"]
	bucketclass := additionalConfig["bucketclass"]

	if err := util.ValidateQuotaConfig(objectName, obcMaxSize, obcMaxObjects); err != nil {
		return err
	}

	if err := validations.ValidateReplicationPolicy(objectName, replicationPolicy, update, isCLI); err != nil {
		return err
	}

	if err := validations.ValidateNSFSAccountConfig(NSFSAccountConfig, bucketclass); err != nil {
		return err
	}

	if err := validateBucketType(additionalConfig); err != nil {
		return err
	}

	return nil
}

func validateBucketType(additionalConfig map[string]string) error {
	bucketType := additionalConfig["bucketType"]
	if bucketType != "" && bucketType != "data" && bucketType != "vector" {
		return fmt.Errorf("invalid bucketType %q, must be 'data' or 'vector'", bucketType)
	}

	vectorDBType := additionalConfig["vectorDBType"]
	if vectorDBType != "" && vectorDBType != "lance" && vectorDBType != "opensearch" && vectorDBType != "davinci" {
		return fmt.Errorf("invalid vectorDBType %q, must be one of: lance, opensearch, davinci", vectorDBType)
	}
	if vectorDBType == "opensearch" {
		return fmt.Errorf("vectorDBType %q is not yet supported", vectorDBType)
	}

	if bucketType == "data" && vectorDBType != "" {
		return fmt.Errorf("vectorDBType %q cannot be set when bucketType is 'data'", vectorDBType)
	}

	lanceConfigJSON := additionalConfig["lanceConfig"]
	if lanceConfigJSON != "" {
		if vectorDBType != "" && vectorDBType != "lance" {
			return fmt.Errorf("lanceConfig cannot be set when vectorDBType is %q", vectorDBType)
		}
		var lanceConfig nb.LanceConfig
		if err := json.Unmarshal([]byte(lanceConfigJSON), &lanceConfig); err != nil {
			return fmt.Errorf("invalid lanceConfig JSON: %v", err)
		}
	}

	return nil
}
