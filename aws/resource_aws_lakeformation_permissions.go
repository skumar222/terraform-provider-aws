package aws

import (
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lakeformation"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/hashcode"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
	tflakeformation "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/lakeformation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/lakeformation/waiter"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func resourceAwsLakeFormationPermissions() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLakeFormationPermissionsCreate,
		Read:   resourceAwsLakeFormationPermissionsRead,
		Delete: resourceAwsLakeFormationPermissionsDelete,

		Schema: map[string]*schema.Schema{
			"catalog_id": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"catalog_resource": {
				Type:     schema.TypeBool,
				Default:  false,
				ForceNew: true,
				Optional: true,
				ExactlyOneOf: []string{
					"catalog_resource",
					"data_location",
					"database",
					"table",
					"table_with_columns",
				},
			},
			"data_location": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"catalog_resource",
					"data_location",
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"arn": {
							Type:         schema.TypeString,
							ForceNew:     true,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: validateAwsAccountId,
						},
					},
				},
			},
			"database": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"catalog_resource",
					"data_location",
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: validateAwsAccountId,
						},
						"name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
					},
				},
			},
			"permissions": {
				Type:     schema.TypeList,
				ForceNew: true,
				MinItems: 1,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(lakeformation.Permission_Values(), false),
				},
			},
			"permissions_with_grant_option": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(lakeformation.Permission_Values(), false),
				},
			},
			"principal": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validatePrincipal,
			},
			"table": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"catalog_resource",
					"data_location",
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: validateAwsAccountId,
						},
						"database_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table.0.name",
								"table.0.wildcard",
							},
						},
						"wildcard": {
							Type:     schema.TypeBool,
							Default:  false,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table.0.name",
								"table.0.wildcard",
							},
						},
					},
				},
			},
			"table_with_columns": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"catalog_resource",
					"data_location",
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: validateAwsAccountId,
						},
						"column_names": {
							Type:     schema.TypeSet,
							ForceNew: true,
							Optional: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.NoZeroValues,
							},
							AtLeastOneOf: []string{
								"table_with_columns.0.column_names",
								"table_with_columns.0.wildcard",
							},
						},
						"database_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"excluded_column_names": {
							Type:     schema.TypeSet,
							ForceNew: true,
							Optional: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.NoZeroValues,
							},
						},
						"name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"wildcard": {
							Type:     schema.TypeBool,
							Default:  false,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table_with_columns.0.column_names",
								"table_with_columns.0.wildcard",
							},
						},
					},
				},
			},
		},
	}
}

// The challenges with Lake Formation permissions are many. These are largely undocumented and
// discovered through trial and error. These are specific problems discovered thus far:
// 1. Implicit permissions granted by Lake Formation to data lake administrators are indistinguishable
//    from explicit permissions. However, implicit permissions cannot be changed, revoked, or narrowed.
// 2. One set of permissions for one LF Resource going in, can come back from AWS as multiple sets of
//    permissions for multiple LF Resources (e.g., SELECT, Table, TableWithColumns).
// 3. Valid permissions for a Table LF resource can come back in TableWithColumns and vice versa.

// For 2 & 3, some peeking at the config (i.e., d.Get()) is necessary to filter the permissions AWS
// returns.

func resourceAwsLakeFormationPermissionsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lakeformationconn

	input := &lakeformation.GrantPermissionsInput{
		Permissions: expandStringList(d.Get("permissions").([]interface{})),
		Principal: &lakeformation.DataLakePrincipal{
			DataLakePrincipalIdentifier: aws.String(d.Get("principal").(string)),
		},
		Resource: &lakeformation.Resource{},
	}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("permissions_with_grant_option"); ok {
		input.PermissionsWithGrantOption = expandStringList(v.([]interface{}))
	}

	if _, ok := d.GetOk("catalog_resource"); ok {
		input.Resource.Catalog = expandLakeFormationCatalogResource()
	}

	if v, ok := d.GetOk("data_location"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.DataLocation = expandLakeFormationDataLocationResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("database"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Database = expandLakeFormationDatabaseResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("table"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Table = expandLakeFormationTableResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("table_with_columns"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.TableWithColumns = expandLakeFormationTableWithColumnsResource(v.([]interface{})[0].(map[string]interface{}))
	}

	var output *lakeformation.GrantPermissionsOutput
	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error
		output, err = conn.GrantPermissions(input)
		if err != nil {
			if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "Invalid principal") {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "Grantee has no permissions") {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "register the S3 path") {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrCodeEquals(err, lakeformation.ErrCodeConcurrentModificationException) {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized to access requested permissions") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(fmt.Errorf("error creating Lake Formation Permissions: %w", err))
		}
		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = conn.GrantPermissions(input)
	}

	if err != nil {
		return fmt.Errorf("error creating Lake Formation Permissions (input: %v): %w", input, err)
	}

	if output == nil {
		return fmt.Errorf("error creating Lake Formation Permissions: empty response")
	}

	d.SetId(fmt.Sprintf("%d", hashcode.String(input.String())))

	return resourceAwsLakeFormationPermissionsRead(d, meta)
}

func resourceAwsLakeFormationPermissionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lakeformationconn

	input := &lakeformation.ListPermissionsInput{
		Principal: &lakeformation.DataLakePrincipal{
			DataLakePrincipalIdentifier: aws.String(d.Get("principal").(string)),
		},
		Resource: &lakeformation.Resource{},
	}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	if _, ok := d.GetOk("catalog_resource"); ok {
		input.Resource.Catalog = expandLakeFormationCatalogResource()
	}

	if v, ok := d.GetOk("data_location"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.DataLocation = expandLakeFormationDataLocationResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("database"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Database = expandLakeFormationDatabaseResource(v.([]interface{})[0].(map[string]interface{}))
	}

	tableType := ""

	if v, ok := d.GetOk("table"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Table = expandLakeFormationTableResource(v.([]interface{})[0].(map[string]interface{}))
		tableType = tflakeformation.TableTypeTable
	}

	if v, ok := d.GetOk("table_with_columns"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		// can't ListPermissions for TableWithColumns, so use Table instead
		input.Resource.Table = expandLakeFormationTableWithColumnsResourceAsTable(v.([]interface{})[0].(map[string]interface{}))
		tableType = tflakeformation.TableTypeTableWithColumns
	}

	columnNames := make([]*string, 0)
	excludedColumnNames := make([]*string, 0)
	columnWildcard := false

	if tableType == tflakeformation.TableTypeTableWithColumns {
		if v, ok := d.GetOk("table_with_columns.0.wildcard"); ok {
			columnWildcard = v.(bool)
		}

		if v, ok := d.GetOk("table_with_columns.0.column_names"); ok {
			if v, ok := v.(*schema.Set); ok && v.Len() > 0 {
				columnNames = expandStringSet(v)
			}
		}

		if v, ok := d.GetOk("table_with_columns.0.excluded_column_names"); ok {
			if v, ok := v.(*schema.Set); ok && v.Len() > 0 {
				excludedColumnNames = expandStringSet(v)
			}
		}
	}

	log.Printf("[DEBUG] Reading Lake Formation permissions: %v", input)

	allPermissions, err := waiter.PermissionsReady(conn, input, tableType, columnNames, excludedColumnNames, columnWildcard)

	if !d.IsNewResource() {
		if tfawserr.ErrCodeEquals(err, lakeformation.ErrCodeEntityNotFoundException) {
			log.Printf("[WARN] Resource Lake Formation permissions (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		if tfawserr.ErrMessageContains(err, "AccessDeniedException", "Resource does not exist") {
			log.Printf("[WARN] Resource Lake Formation permissions (%s) not found, removing from state: %s", d.Id(), err)
			d.SetId("")
			return nil
		}

		if len(allPermissions) == 0 {
			log.Printf("[WARN] Resource Lake Formation permissions (%s) not found, removing from state (0 permissions)", d.Id())
			d.SetId("")
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("error reading Lake Formation permissions: %w", err)
	}

	// clean permissions = filter out permissions that do not pertain to this specific resource
	cleanPermissions := tflakeformation.FilterPermissions(input, tableType, columnNames, excludedColumnNames, columnWildcard, allPermissions)

	if len(cleanPermissions) == 0 {
		log.Printf("[WARN] No Lake Formation permissions (%s) found", d.Id())
		d.Set("catalog_resource", false)
		d.Set("data_location", nil)
		d.Set("database", nil)
		d.Set("table_with_columns", nil)
		d.Set("table", nil)
		return nil
	}

	if len(cleanPermissions) != len(allPermissions) {
		log.Printf("[INFO] Resource Lake Formation clean permissions (%d) and all permissions (%d) have different lengths (this is not necessarily a problem): %s", len(cleanPermissions), len(allPermissions), d.Id())
	}

	d.Set("principal", cleanPermissions[0].Principal.DataLakePrincipalIdentifier)
	d.Set("permissions", flattenLakeFormationPermissions(cleanPermissions))
	d.Set("permissions_with_grant_option", flattenLakeFormationGrantPermissions(cleanPermissions))

	if cleanPermissions[0].Resource.Catalog != nil {
		d.Set("catalog_resource", true)
	} else {
		d.Set("catalog_resource", false)
	}

	if cleanPermissions[0].Resource.DataLocation != nil {
		if err := d.Set("data_location", []interface{}{flattenLakeFormationDataLocationResource(cleanPermissions[0].Resource.DataLocation)}); err != nil {
			return fmt.Errorf("error setting data_location: %w", err)
		}
	} else {
		d.Set("data_location", nil)
	}

	if cleanPermissions[0].Resource.Database != nil {
		if err := d.Set("database", []interface{}{flattenLakeFormationDatabaseResource(cleanPermissions[0].Resource.Database)}); err != nil {
			return fmt.Errorf("error setting database: %w", err)
		}
	} else {
		d.Set("database", nil)
	}

	tableSet := false

	if v, ok := d.GetOk("table"); ok && len(v.([]interface{})) > 0 {
		// since perm list could include TableWithColumns, get the right one
		for _, perm := range cleanPermissions {
			if perm.Resource == nil {
				continue
			}

			if perm.Resource.TableWithColumns != nil && perm.Resource.TableWithColumns.ColumnWildcard != nil {
				if err := d.Set("table", []interface{}{flattenLakeFormationTableWithColumnsResourceAsTable(perm.Resource.TableWithColumns)}); err != nil {
					return fmt.Errorf("error setting table: %w", err)
				}
				tableSet = true
				break
			}

			if perm.Resource.Table != nil {
				if err := d.Set("table", []interface{}{flattenLakeFormationTableResource(perm.Resource.Table)}); err != nil {
					return fmt.Errorf("error setting table: %w", err)
				}
				tableSet = true
				break
			}
		}
	}

	if !tableSet {
		d.Set("table", nil)
	}

	twcSet := false

	if v, ok := d.GetOk("table_with_columns"); ok && len(v.([]interface{})) > 0 {
		// since perm list could include Table, get the right one
		for _, perm := range cleanPermissions {
			if perm.Resource.TableWithColumns != nil {
				if err := d.Set("table_with_columns", []interface{}{flattenLakeFormationTableWithColumnsResource(perm.Resource.TableWithColumns)}); err != nil {
					return fmt.Errorf("error setting table_with_columns: %w", err)
				}
				twcSet = true
				break
			}
		}
	}

	if !twcSet {
		d.Set("table_with_columns", nil)
	}

	return nil
}

func resourceAwsLakeFormationPermissionsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lakeformationconn

	input := &lakeformation.RevokePermissionsInput{
		Permissions:                expandStringList(d.Get("permissions").([]interface{})),
		PermissionsWithGrantOption: expandStringList(d.Get("permissions_with_grant_option").([]interface{})),
		Principal: &lakeformation.DataLakePrincipal{
			DataLakePrincipalIdentifier: aws.String(d.Get("principal").(string)),
		},
		Resource: &lakeformation.Resource{},
	}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	if _, ok := d.GetOk("catalog_resource"); ok {
		input.Resource.Catalog = expandLakeFormationCatalogResource()
	}

	if v, ok := d.GetOk("data_location"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.DataLocation = expandLakeFormationDataLocationResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("database"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Database = expandLakeFormationDatabaseResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("table"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.Table = expandLakeFormationTableResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("table_with_columns"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Resource.TableWithColumns = expandLakeFormationTableWithColumnsResource(v.([]interface{})[0].(map[string]interface{}))
	}

	if input.Resource == nil || reflect.DeepEqual(input.Resource, &lakeformation.Resource{}) {
		// if resource is empty, don't delete = it won't delete anything since this is the predicate
		log.Printf("[WARN] No Lake Formation Resource with permissions to revoke")
		return nil
	}

	err := resource.Retry(waiter.PermissionsDeleteRetryTimeout, func() *resource.RetryError {
		var err error
		_, err = conn.RevokePermissions(input)
		if err != nil {
			if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "register the S3 path") {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrCodeEquals(err, lakeformation.ErrCodeConcurrentModificationException) {
				return resource.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized to access requested permissions") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(fmt.Errorf("unable to revoke Lake Formation Permissions: %w", err))
		}
		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = conn.RevokePermissions(input)
	}

	if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "No permissions revoked. Grantee") {
		return nil
	}

	if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "cannot grant/revoke permission on non-existent column") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("unable to revoke LakeFormation Permissions (input: %v): %w", input, err)
	}

	// Attempted to add a waiter here to wait for delete to complete. However, ListPermissions() returns
	// permissions, at least for catalog/CREATE_DATABASE permission, even if they do not exist. That makes
	// knowing when the delete is complete impossible. Instead, we'll retry until we get the right error.

	// Knowing when the delete is complete is complicated:
	// You can't just wait until permissions = 0 because there could be many other unrelated permissions
	// on the resource and filtering is non-trivial for table with columns.

	err = resource.Retry(waiter.PermissionsDeleteRetryTimeout, func() *resource.RetryError {
		var err error
		_, err = conn.RevokePermissions(input)

		if !tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "No permissions revoked. Grantee has no") {
			return resource.RetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = conn.RevokePermissions(input)
	}

	if tfawserr.ErrMessageContains(err, lakeformation.ErrCodeInvalidInputException, "No permissions revoked. Grantee") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("unable to revoke LakeFormation Permissions (input: %v): %w", input, err)
	}

	return nil
}

func expandLakeFormationCatalogResource() *lakeformation.CatalogResource {
	return &lakeformation.CatalogResource{}
}

func expandLakeFormationDataLocationResource(tfMap map[string]interface{}) *lakeformation.DataLocationResource {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.DataLocationResource{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["arn"].(string); ok && v != "" {
		apiObject.ResourceArn = aws.String(v)
	}

	return apiObject
}

func flattenLakeFormationDataLocationResource(apiObject *lakeformation.DataLocationResource) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	if v := apiObject.ResourceArn; v != nil {
		tfMap["arn"] = aws.StringValue(v)
	}

	return tfMap
}

func expandLakeFormationDatabaseResource(tfMap map[string]interface{}) *lakeformation.DatabaseResource {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.DatabaseResource{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["name"].(string); ok && v != "" {
		apiObject.Name = aws.String(v)
	}

	return apiObject
}

func flattenLakeFormationDatabaseResource(apiObject *lakeformation.DatabaseResource) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	if v := apiObject.Name; v != nil {
		tfMap["name"] = aws.StringValue(v)
	}

	return tfMap
}

func expandLakeFormationTableResource(tfMap map[string]interface{}) *lakeformation.TableResource {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.TableResource{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["database_name"].(string); ok && v != "" {
		apiObject.DatabaseName = aws.String(v)
	}

	if v, ok := tfMap["name"].(string); ok && v != "" {
		apiObject.Name = aws.String(v)
	}

	if v, ok := tfMap["wildcard"].(bool); ok && v {
		apiObject.TableWildcard = &lakeformation.TableWildcard{}
	}

	return apiObject
}

func expandLakeFormationTableWithColumnsResourceAsTable(tfMap map[string]interface{}) *lakeformation.TableResource {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.TableResource{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["database_name"].(string); ok && v != "" {
		apiObject.DatabaseName = aws.String(v)
	}

	if v, ok := tfMap["name"].(string); ok && v != "" {
		apiObject.Name = aws.String(v)
	}

	return apiObject
}

func flattenLakeFormationTableResource(apiObject *lakeformation.TableResource) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	if v := apiObject.DatabaseName; v != nil {
		tfMap["database_name"] = aws.StringValue(v)
	}

	if v := apiObject.Name; v != nil {
		if aws.StringValue(v) != tflakeformation.TableNameAllTables || apiObject.TableWildcard == nil {
			tfMap["name"] = aws.StringValue(v)
		}
	}

	if v := apiObject.TableWildcard; v != nil {
		tfMap["wildcard"] = true
	}

	return tfMap
}

func expandLakeFormationTableWithColumnsResource(tfMap map[string]interface{}) *lakeformation.TableWithColumnsResource {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.TableWithColumnsResource{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["column_names"]; ok {
		if v, ok := v.(*schema.Set); ok && v.Len() > 0 {
			apiObject.ColumnNames = expandStringSet(v)
		}
	}

	if v, ok := tfMap["database_name"].(string); ok && v != "" {
		apiObject.DatabaseName = aws.String(v)
	}

	if v, ok := tfMap["excluded_column_names"]; ok {
		if v, ok := v.(*schema.Set); ok && v.Len() > 0 {
			apiObject.ColumnWildcard = &lakeformation.ColumnWildcard{
				ExcludedColumnNames: expandStringSet(v),
			}
		}
	}

	if v, ok := tfMap["name"].(string); ok && v != "" {
		apiObject.Name = aws.String(v)
	}

	if v, ok := tfMap["wildcard"].(bool); ok && v && apiObject.ColumnWildcard == nil {
		apiObject.ColumnWildcard = &lakeformation.ColumnWildcard{}
	}

	return apiObject
}

func flattenLakeFormationTableWithColumnsResource(apiObject *lakeformation.TableWithColumnsResource) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	tfMap["column_names"] = flattenStringSet(apiObject.ColumnNames)

	if v := apiObject.DatabaseName; v != nil {
		tfMap["database_name"] = aws.StringValue(v)
	}

	if v := apiObject.ColumnWildcard; v != nil {
		tfMap["wildcard"] = true
		tfMap["excluded_column_names"] = flattenStringSet(v.ExcludedColumnNames)
	}

	if v := apiObject.Name; v != nil {
		tfMap["name"] = aws.StringValue(v)
	}

	return tfMap
}

// This only happens in very specific situations:
// (Select) TWC + ColumnWildcard              = (Select) Table
// (Select) TWC + ColumnWildcard + ALL_TABLES = (Select) Table + TableWildcard
func flattenLakeFormationTableWithColumnsResourceAsTable(apiObject *lakeformation.TableWithColumnsResource) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	if v := apiObject.DatabaseName; v != nil {
		tfMap["database_name"] = aws.StringValue(v)
	}

	if v := apiObject.Name; v != nil && aws.StringValue(v) == tflakeformation.TableNameAllTables && apiObject.ColumnWildcard != nil {
		tfMap["wildcard"] = true
	} else if v := apiObject.Name; v != nil {
		tfMap["name"] = aws.StringValue(v)
	}

	return tfMap
}

func flattenLakeFormationPermissions(apiObjects []*lakeformation.PrincipalResourcePermissions) []string {
	if apiObjects == nil {
		return nil
	}

	tfList := make([]string, 0)

	for _, resourcePermission := range apiObjects {
		for _, permission := range resourcePermission.Permissions {
			tfList = append(tfList, aws.StringValue(permission))
		}
	}

	return tfList
}

func flattenLakeFormationGrantPermissions(apiObjects []*lakeformation.PrincipalResourcePermissions) []string {
	if apiObjects == nil {
		return nil
	}

	tfList := make([]string, 0)

	for _, resourcePermission := range apiObjects {
		for _, grantPermission := range resourcePermission.PermissionsWithGrantOption {
			tfList = append(tfList, aws.StringValue(grantPermission))
		}
	}

	return tfList
}
