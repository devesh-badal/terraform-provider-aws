package s3

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceBucketWebsiteConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketWebsiteConfigurationCreate,
		ReadContext:   resourceBucketWebsiteConfigurationRead,
		UpdateContext: resourceBucketWebsiteConfigurationUpdate,
		DeleteContext: resourceBucketWebsiteConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 63),
			},
			"error_document": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"expected_bucket_owner": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidAccountID,
			},
			"index_document": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"suffix": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"redirect_all_requests_to": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ConflictsWith: []string{
					"error_document",
					"index_document",
					"routing_rule",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice(s3.Protocol_Values(), false),
						},
					},
				},
			},
			"routing_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"condition": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"http_error_code_returned_equals": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"key_prefix_equals": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"redirect": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"http_redirect_code": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"protocol": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringInSlice(s3.Protocol_Values(), false),
									},
									"replace_key_prefix_with": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"replace_key_with": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceBucketWebsiteConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	bucket := d.Get("bucket").(string)
	expectedBucketOwner := d.Get("expected_bucket_owner").(string)

	websiteConfig := &s3.WebsiteConfiguration{}

	if v, ok := d.GetOk("error_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.ErrorDocument = expandS3BucketWebsiteConfigurationErrorDocument(v.([]interface{}))
	}

	if v, ok := d.GetOk("index_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.IndexDocument = expandS3BucketWebsiteConfigurationIndexDocument(v.([]interface{}))
	}

	if v, ok := d.GetOk("redirect_all_requests_to"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.RedirectAllRequestsTo = expandS3BucketWebsiteConfigurationRedirectAllRequestsTo(v.([]interface{}))
	}

	if v, ok := d.GetOk("routing_rule"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.RoutingRules = expandS3BucketWebsiteConfigurationRoutingRules(v.([]interface{}))
	}

	input := &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucket),
		WebsiteConfiguration: websiteConfig,
	}

	if expectedBucketOwner != "" {
		input.ExpectedBucketOwner = aws.String(expectedBucketOwner)
	}

	_, err := verify.RetryOnAWSCode(s3.ErrCodeNoSuchBucket, func() (interface{}, error) {
		return conn.PutBucketWebsiteWithContext(ctx, input)
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating S3 bucket (%s) website configuration: %w", bucket, err))
	}

	d.SetId(resourceBucketWebsiteConfigurationCreateResourceID(bucket, expectedBucketOwner))

	return resourceBucketWebsiteConfigurationRead(ctx, d, meta)
}

func resourceBucketWebsiteConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	bucket, expectedBucketOwner, err := resourceBucketWebsiteConfigurationParseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	input := &s3.GetBucketWebsiteInput{
		Bucket: aws.String(bucket),
	}

	if expectedBucketOwner != "" {
		input.ExpectedBucketOwner = aws.String(expectedBucketOwner)
	}

	output, err := conn.GetBucketWebsiteWithContext(ctx, input)

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, s3.ErrCodeNoSuchBucket, ErrCodeNoSuchWebsiteConfiguration) {
		log.Printf("[WARN] S3 Bucket Website Configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if output == nil {
		if d.IsNewResource() {
			return diag.FromErr(fmt.Errorf("error reading S3 bucket website configuration (%s): empty output", d.Id()))
		}
		log.Printf("[WARN] S3 Bucket Website Configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("bucket", bucket)
	d.Set("expected_bucket_owner", expectedBucketOwner)

	if err := d.Set("error_document", flattenS3BucketWebsiteConfigurationErrorDocument(output.ErrorDocument)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting error_document: %w", err))
	}

	if err := d.Set("index_document", flattenS3BucketWebsiteConfigurationIndexDocument(output.IndexDocument)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting index_document: %w", err))
	}

	if err := d.Set("redirect_all_requests_to", flattenS3BucketWebsiteConfigurationRedirectAllRequestsTo(output.RedirectAllRequestsTo)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting redirect_all_requests_to: %w", err))
	}

	if err := d.Set("routing_rule", flattenS3BucketWebsiteConfigurationRoutingRules(output.RoutingRules)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting routing_rule: %w", err))
	}

	return nil
}

func resourceBucketWebsiteConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	bucket, expectedBucketOwner, err := resourceBucketWebsiteConfigurationParseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	websiteConfig := &s3.WebsiteConfiguration{}

	if v, ok := d.GetOk("error_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.ErrorDocument = expandS3BucketWebsiteConfigurationErrorDocument(v.([]interface{}))
	}

	if v, ok := d.GetOk("index_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.IndexDocument = expandS3BucketWebsiteConfigurationIndexDocument(v.([]interface{}))
	}

	if v, ok := d.GetOk("redirect_all_requests_to"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.RedirectAllRequestsTo = expandS3BucketWebsiteConfigurationRedirectAllRequestsTo(v.([]interface{}))
	}

	if v, ok := d.GetOk("routing_rule"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.RoutingRules = expandS3BucketWebsiteConfigurationRoutingRules(v.([]interface{}))
	}

	input := &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucket),
		WebsiteConfiguration: websiteConfig,
	}

	if expectedBucketOwner != "" {
		input.ExpectedBucketOwner = aws.String(expectedBucketOwner)
	}

	_, err = conn.PutBucketWebsiteWithContext(ctx, input)

	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating S3 bucket website configuration (%s): %w", d.Id(), err))
	}

	return resourceBucketWebsiteConfigurationRead(ctx, d, meta)
}

func resourceBucketWebsiteConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	bucket, expectedBucketOwner, err := resourceBucketWebsiteConfigurationParseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	input := &s3.DeleteBucketWebsiteInput{
		Bucket: aws.String(bucket),
	}

	if expectedBucketOwner != "" {
		input.ExpectedBucketOwner = aws.String(expectedBucketOwner)
	}

	_, err = conn.DeleteBucketWebsiteWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, s3.ErrCodeNoSuchBucket, ErrCodeNoSuchWebsiteConfiguration) {
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting S3 bucket website configuration (%s): %w", d.Id(), err))
	}

	return nil
}

func resourceBucketWebsiteConfigurationCreateResourceID(bucket, expectedBucketOwner string) string {
	if bucket == "" {
		return expectedBucketOwner
	}

	if expectedBucketOwner == "" {
		return bucket
	}

	parts := []string{bucket, expectedBucketOwner}
	id := strings.Join(parts, ",")

	return id
}

func resourceBucketWebsiteConfigurationParseResourceID(id string) (string, string, error) {
	parts := strings.Split(id, ",")

	if len(parts) == 1 && parts[0] != "" {
		return parts[0], "", nil
	}

	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected format for ID (%[1]s), expected BUCKET or BUCKET,EXPECTED_BUCKET_OWNER", id)
}

func expandS3BucketWebsiteConfigurationErrorDocument(l []interface{}) *s3.ErrorDocument {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.ErrorDocument{}

	if v, ok := tfMap["key"].(string); ok && v != "" {
		result.Key = aws.String(v)
	}

	return result
}

func expandS3BucketWebsiteConfigurationIndexDocument(l []interface{}) *s3.IndexDocument {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.IndexDocument{}

	if v, ok := tfMap["suffix"].(string); ok && v != "" {
		result.Suffix = aws.String(v)
	}

	return result
}

func expandS3BucketWebsiteConfigurationRedirectAllRequestsTo(l []interface{}) *s3.RedirectAllRequestsTo {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.RedirectAllRequestsTo{}

	if v, ok := tfMap["host_name"].(string); ok && v != "" {
		result.HostName = aws.String(v)
	}

	if v, ok := tfMap["protocol"].(string); ok && v != "" {
		result.Protocol = aws.String(v)
	}

	return result
}

func expandS3BucketWebsiteConfigurationRoutingRules(l []interface{}) []*s3.RoutingRule {
	var results []*s3.RoutingRule

	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}

		rule := &s3.RoutingRule{}

		if v, ok := tfMap["condition"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
			rule.Condition = expandS3BucketWebsiteConfigurationRoutingRuleCondition(v)
		}

		if v, ok := tfMap["redirect"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
			rule.Redirect = expandS3BucketWebsiteConfigurationRoutingRuleRedirect(v)
		}

		results = append(results, rule)
	}

	return results
}

func expandS3BucketWebsiteConfigurationRoutingRuleCondition(l []interface{}) *s3.Condition {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.Condition{}

	if v, ok := tfMap["http_error_code_returned_equals"].(string); ok && v != "" {
		result.HttpErrorCodeReturnedEquals = aws.String(v)
	}

	if v, ok := tfMap["key_prefix_equals"].(string); ok && v != "" {
		result.KeyPrefixEquals = aws.String(v)
	}

	return result
}

func expandS3BucketWebsiteConfigurationRoutingRuleRedirect(l []interface{}) *s3.Redirect {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.Redirect{}

	if v, ok := tfMap["host_name"].(string); ok && v != "" {
		result.HostName = aws.String(v)
	}

	if v, ok := tfMap["http_redirect_code"].(string); ok && v != "" {
		result.HttpRedirectCode = aws.String(v)
	}

	if v, ok := tfMap["protocol"].(string); ok && v != "" {
		result.Protocol = aws.String(v)
	}

	if v, ok := tfMap["replace_key_prefix_with"].(string); ok && v != "" {
		result.ReplaceKeyPrefixWith = aws.String(v)
	}

	if v, ok := tfMap["replace_key_with"].(string); ok && v != "" {
		result.ReplaceKeyWith = aws.String(v)
	}

	return result
}

func flattenS3BucketWebsiteConfigurationIndexDocument(i *s3.IndexDocument) []interface{} {
	if i == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if i.Suffix != nil {
		m["suffix"] = aws.StringValue(i.Suffix)
	}

	return []interface{}{m}
}

func flattenS3BucketWebsiteConfigurationErrorDocument(e *s3.ErrorDocument) []interface{} {
	if e == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if e.Key != nil {
		m["key"] = aws.StringValue(e.Key)
	}

	return []interface{}{m}
}

func flattenS3BucketWebsiteConfigurationRedirectAllRequestsTo(r *s3.RedirectAllRequestsTo) []interface{} {
	if r == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if r.HostName != nil {
		m["host_name"] = aws.StringValue(r.HostName)
	}

	if r.Protocol != nil {
		m["protocol"] = aws.StringValue(r.Protocol)
	}

	return []interface{}{m}
}

func flattenS3BucketWebsiteConfigurationRoutingRules(rules []*s3.RoutingRule) []interface{} {
	var results []interface{}

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		m := make(map[string]interface{})

		if rule.Condition != nil {
			m["condition"] = flattenS3BucketWebsiteConfigurationRoutingRuleCondition(rule.Condition)
		}

		if rule.Redirect != nil {
			m["redirect"] = flattenS3BucketWebsiteConfigurationRoutingRuleRedirect(rule.Redirect)
		}

		results = append(results, m)
	}

	return results
}

func flattenS3BucketWebsiteConfigurationRoutingRuleCondition(c *s3.Condition) []interface{} {
	if c == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if c.KeyPrefixEquals != nil {
		m["key_prefix_equals"] = aws.StringValue(c.KeyPrefixEquals)
	}

	if c.HttpErrorCodeReturnedEquals != nil {
		m["http_error_code_returned_equals"] = aws.StringValue(c.HttpErrorCodeReturnedEquals)
	}

	return []interface{}{m}
}

func flattenS3BucketWebsiteConfigurationRoutingRuleRedirect(r *s3.Redirect) []interface{} {
	if r == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if r.HostName != nil {
		m["host_name"] = aws.StringValue(r.HostName)
	}

	if r.HttpRedirectCode != nil {
		m["http_redirect_code"] = aws.StringValue(r.HttpRedirectCode)
	}

	if r.Protocol != nil {
		m["protocol"] = aws.StringValue(r.Protocol)
	}

	if r.ReplaceKeyWith != nil {
		m["replace_key_with"] = aws.StringValue(r.ReplaceKeyWith)
	}

	if r.ReplaceKeyPrefixWith != nil {
		m["replace_key_prefix_with"] = aws.StringValue(r.ReplaceKeyPrefixWith)
	}

	return []interface{}{m}
}
