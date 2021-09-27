package waiter

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	tfglue "github.com/hashicorp/terraform-provider-aws/aws/internal/service/glue"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

const (
	// Maximum amount of time to wait for an Operation to return Deleted
	MLTransformDeleteTimeout      = 2 * time.Minute
	RegistryDeleteTimeout         = 2 * time.Minute
	SchemaAvailableTimeout        = 2 * time.Minute
	SchemaDeleteTimeout           = 2 * time.Minute
	SchemaVersionAvailableTimeout = 2 * time.Minute
	TriggerCreateTimeout          = 5 * time.Minute
	TriggerDeleteTimeout          = 5 * time.Minute
)

// MLTransformDeleted waits for an MLTransform to return Deleted
func MLTransformDeleted(conn *glue.Glue, transformId string) (*glue.GetMLTransformOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.TransformStatusTypeNotReady, glue.TransformStatusTypeReady, glue.TransformStatusTypeDeleting},
		Target:  []string{},
		Refresh: MLTransformStatus(conn, transformId),
		Timeout: MLTransformDeleteTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetMLTransformOutput); ok {
		return output, err
	}

	return nil, err
}

// RegistryDeleted waits for a Registry to return Deleted
func RegistryDeleted(conn *glue.Glue, registryID string) (*glue.GetRegistryOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.RegistryStatusDeleting},
		Target:  []string{},
		Refresh: RegistryStatus(conn, registryID),
		Timeout: RegistryDeleteTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetRegistryOutput); ok {
		return output, err
	}

	return nil, err
}

// SchemaAvailable waits for a Schema to return Available
func SchemaAvailable(conn *glue.Glue, registryID string) (*glue.GetSchemaOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.SchemaStatusPending},
		Target:  []string{glue.SchemaStatusAvailable},
		Refresh: SchemaStatus(conn, registryID),
		Timeout: SchemaAvailableTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetSchemaOutput); ok {
		return output, err
	}

	return nil, err
}

// SchemaDeleted waits for a Schema to return Deleted
func SchemaDeleted(conn *glue.Glue, registryID string) (*glue.GetSchemaOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.SchemaStatusDeleting},
		Target:  []string{},
		Refresh: SchemaStatus(conn, registryID),
		Timeout: SchemaDeleteTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetSchemaOutput); ok {
		return output, err
	}

	return nil, err
}

// SchemaVersionAvailable waits for a Schema to return Available
func SchemaVersionAvailable(conn *glue.Glue, registryID string) (*glue.GetSchemaVersionOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.SchemaVersionStatusPending},
		Target:  []string{glue.SchemaVersionStatusAvailable},
		Refresh: SchemaVersionStatus(conn, registryID),
		Timeout: SchemaVersionAvailableTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetSchemaVersionOutput); ok {
		return output, err
	}

	return nil, err
}

// TriggerCreated waits for a Trigger to return Created
func TriggerCreated(conn *glue.Glue, triggerName string) (*glue.GetTriggerOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			glue.TriggerStateActivating,
			glue.TriggerStateCreating,
			glue.TriggerStateUpdating,
		},
		Target: []string{
			glue.TriggerStateActivated,
			glue.TriggerStateCreated,
		},
		Refresh: TriggerStatus(conn, triggerName),
		Timeout: TriggerCreateTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetTriggerOutput); ok {
		return output, err
	}

	return nil, err
}

// TriggerDeleted waits for a Trigger to return Deleted
func TriggerDeleted(conn *glue.Glue, triggerName string) (*glue.GetTriggerOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.TriggerStateDeleting},
		Target:  []string{},
		Refresh: TriggerStatus(conn, triggerName),
		Timeout: TriggerDeleteTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.GetTriggerOutput); ok {
		return output, err
	}

	return nil, err
}

func GlueDevEndpointCreated(conn *glue.Glue, name string) (*glue.DevEndpoint, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{tfglue.DevEndpointStatusProvisioning},
		Target:  []string{tfglue.DevEndpointStatusReady},
		Refresh: GlueDevEndpointStatus(conn, name),
		Timeout: 15 * time.Minute,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.DevEndpoint); ok {
		if status := aws.StringValue(output.Status); status == tfglue.DevEndpointStatusFailed {
			tfresource.SetLastError(err, errors.New(aws.StringValue(output.FailureReason)))
		}

		return output, err
	}

	return nil, err
}

func GlueDevEndpointDeleted(conn *glue.Glue, name string) (*glue.DevEndpoint, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{tfglue.DevEndpointStatusTerminating},
		Target:  []string{},
		Refresh: GlueDevEndpointStatus(conn, name),
		Timeout: 15 * time.Minute,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*glue.DevEndpoint); ok {
		if status := aws.StringValue(output.Status); status == tfglue.DevEndpointStatusFailed {
			tfresource.SetLastError(err, errors.New(aws.StringValue(output.FailureReason)))
		}

		return output, err
	}

	return nil, err
}
