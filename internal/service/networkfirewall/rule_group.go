package networkfirewall

import (
	"context"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_networkfirewall_rule_group", name="Rule Group")
// @Tags(identifierAttribute="id")
func ResourceRuleGroup() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRuleGroupCreate,
		ReadWithoutTimeout:   resourceRuleGroupRead,
		UpdateWithoutTimeout: resourceRuleGroupUpdate,
		DeleteWithoutTimeout: resourceRuleGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"capacity": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"encryption_configuration": encryptionConfigurationSchema(),
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule_group": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"reference_sets": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip_set_references": {
										Type:     schema.TypeSet,
										Optional: true,
										MaxItems: 5,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_set_reference": {
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"reference_arn": {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: verify.ValidARN,
															},
														},
													},
												},
												"key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 32),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z]`), "must begin with alphabetic character"),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_]+$`), "must contain only alphanumeric and underscore characters"),
													),
												},
											},
										},
									},
								},
							},
						},
						"rules_source": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"rules_source_list": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"generated_rules_type": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringInSlice(networkfirewall.GeneratedRulesType_Values(), false),
												},
												"target_types": {
													Type:     schema.TypeSet,
													Required: true,
													Elem: &schema.Schema{
														Type:         schema.TypeString,
														ValidateFunc: validation.StringInSlice(networkfirewall.TargetType_Values(), false),
													},
												},
												"targets": {
													Type:     schema.TypeSet,
													Required: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"rules_string": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"stateful_rule": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"action": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringInSlice(networkfirewall.StatefulAction_Values(), false),
												},
												"header": {
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"destination": {
																Type:     schema.TypeString,
																Required: true,
															},
															"destination_port": {
																Type:     schema.TypeString,
																Required: true,
															},
															"direction": {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: validation.StringInSlice(networkfirewall.StatefulRuleDirection_Values(), false),
															},
															"protocol": {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: validation.StringInSlice(networkfirewall.StatefulRuleProtocol_Values(), false),
															},
															"source": {
																Type:     schema.TypeString,
																Required: true,
															},
															"source_port": {
																Type:     schema.TypeString,
																Required: true,
															},
														},
													},
												},
												"rule_option": {
													Type:     schema.TypeSet,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"keyword": {
																Type:     schema.TypeString,
																Required: true,
															},
															"settings": {
																Type:     schema.TypeSet,
																Optional: true,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
											},
										},
									},
									"stateless_rules_and_custom_actions": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"custom_action": customActionSchema(),
												"stateless_rule": {
													Type:     schema.TypeSet,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"priority": {
																Type:     schema.TypeInt,
																Required: true,
															},
															"rule_definition": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Required: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"actions": {
																			Type:     schema.TypeSet,
																			Required: true,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																		"match_attributes": {
																			Type:     schema.TypeList,
																			MaxItems: 1,
																			Required: true,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{
																					"destination": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem: &schema.Resource{
																							Schema: map[string]*schema.Schema{
																								"address_definition": {
																									Type:         schema.TypeString,
																									Required:     true,
																									ValidateFunc: verify.ValidIPv4CIDRNetworkAddress,
																								},
																							},
																						},
																					},
																					"destination_port": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem: &schema.Resource{
																							Schema: map[string]*schema.Schema{
																								"from_port": {
																									Type:     schema.TypeInt,
																									Required: true,
																								},
																								"to_port": {
																									Type:     schema.TypeInt,
																									Optional: true,
																								},
																							},
																						},
																					},
																					"protocols": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem:     &schema.Schema{Type: schema.TypeInt},
																					},
																					"source": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem: &schema.Resource{
																							Schema: map[string]*schema.Schema{
																								"address_definition": {
																									Type:         schema.TypeString,
																									Required:     true,
																									ValidateFunc: verify.ValidIPv4CIDRNetworkAddress,
																								},
																							},
																						},
																					},
																					"source_port": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem: &schema.Resource{
																							Schema: map[string]*schema.Schema{
																								"from_port": {
																									Type:     schema.TypeInt,
																									Required: true,
																								},
																								"to_port": {
																									Type:     schema.TypeInt,
																									Optional: true,
																								},
																							},
																						},
																					},
																					"tcp_flag": {
																						Type:     schema.TypeSet,
																						Optional: true,
																						Elem: &schema.Resource{
																							Schema: map[string]*schema.Schema{
																								"flags": {
																									Type:     schema.TypeSet,
																									Required: true,
																									Elem: &schema.Schema{
																										Type:         schema.TypeString,
																										ValidateFunc: validation.StringInSlice(networkfirewall.TCPFlag_Values(), false),
																									},
																								},
																								"masks": {
																									Type:     schema.TypeSet,
																									Optional: true,
																									Elem: &schema.Schema{
																										Type:         schema.TypeString,
																										ValidateFunc: validation.StringInSlice(networkfirewall.TCPFlag_Values(), false),
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"rule_variables": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip_sets": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 32),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z]`), "must begin with alphabetic character"),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_]+$`), "must contain only alphanumeric and underscore characters"),
													),
												},
												"ip_set": {
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"definition": {
																Type:     schema.TypeSet,
																Required: true,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
											},
										},
									},
									"port_sets": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 32),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z]`), "must begin with alphabetic character"),
														validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_]+$`), "must contain only alphanumeric and underscore characters"),
													),
												},
												"port_set": {
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"definition": {
																Type:     schema.TypeSet,
																Required: true,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"stateful_rule_options": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"rule_order": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(networkfirewall.RuleOrder_Values(), false),
									},
								},
							},
						},
					},
				},
			},
			"rules": {
				Type:     schema.TypeString,
				Optional: true,
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(networkfirewall.RuleGroupType_Values(), false),
			},
			"update_token": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		CustomizeDiff: customdiff.Sequence(
			// The stateful rule_order default action can be explicitly or implicitly set,
			// so ignore spurious diffs if toggling between the two.
			func(_ context.Context, d *schema.ResourceDiff, meta interface{}) error {
				return forceNewIfNotRuleOrderDefault("rule_group.0.stateful_rule_options.0.rule_order", d)
			},
			verify.SetTagsDiff,
		),
	}
}

func resourceRuleGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).NetworkFirewallConn()

	name := d.Get("name").(string)
	input := &networkfirewall.CreateRuleGroupInput{
		Capacity:      aws.Int64(int64(d.Get("capacity").(int))),
		RuleGroupName: aws.String(name),
		Tags:          GetTagsIn(ctx),
		Type:          aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("encryption_configuration"); ok {
		input.EncryptionConfiguration = expandEncryptionConfiguration(v.([]interface{}))
	}

	if v, ok := d.GetOk("rule_group"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.RuleGroup = expandRuleGroup(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("rules"); ok {
		input.Rules = aws.String(v.(string))
	}

	output, err := conn.CreateRuleGroupWithContext(ctx, input)

	if err != nil {
		return diag.Errorf("creating NetworkFirewall Rule Group (%s): %s", name, err)
	}

	d.SetId(aws.StringValue(output.RuleGroupResponse.RuleGroupArn))

	return resourceRuleGroupRead(ctx, d, meta)
}

func resourceRuleGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).NetworkFirewallConn()

	output, err := FindRuleGroupByARN(ctx, conn, d.Id())

	if err == nil && output.RuleGroup == nil {
		err = tfresource.NewEmptyResultError(d.Id())
	}

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] NetworkFirewall Rule Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.Errorf("reading NetworkFirewall Rule Group (%s): %s", d.Id(), err)
	}

	response := output.RuleGroupResponse
	d.Set("arn", response.RuleGroupArn)
	d.Set("capacity", response.Capacity)
	d.Set("description", response.Description)
	d.Set("encryption_configuration", flattenEncryptionConfiguration(response.EncryptionConfiguration))
	d.Set("name", response.RuleGroupName)
	if err := d.Set("rule_group", flattenRuleGroup(output.RuleGroup)); err != nil {
		return diag.Errorf("setting rule_group: %s", err)
	}
	d.Set("type", response.Type)
	d.Set("update_token", output.UpdateToken)

	SetTagsOut(ctx, response.Tags)

	return nil
}

func resourceRuleGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).NetworkFirewallConn()

	if d.HasChanges("description", "encryption_configuration", "rule_group", "rules", "type") {
		input := &networkfirewall.UpdateRuleGroupInput{
			EncryptionConfiguration: expandEncryptionConfiguration(d.Get("encryption_configuration").([]interface{})),
			RuleGroupArn:            aws.String(d.Id()),
			Type:                    aws.String(d.Get("type").(string)),
			UpdateToken:             aws.String(d.Get("update_token").(string)),
		}

		if v, ok := d.GetOk("description"); ok {
			input.Description = aws.String(v.(string))
		}

		// Network Firewall UpdateRuleGroup API method only allows one of Rules or RuleGroup
		// else, request returns "InvalidRequestException: Exactly one of Rules or RuleGroup must be set";
		// Here, "rules" takes precedence as "rule_group" is Computed from "rules" when configured
		// Reference: https://github.com/hashicorp/terraform-provider-aws/issues/19414
		if d.HasChange("rules") {
			input.Rules = aws.String(d.Get("rules").(string))
		} else if d.HasChange("rule_group") {
			if v, ok := d.GetOk("rule_group"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
				input.RuleGroup = expandRuleGroup(v.([]interface{})[0].(map[string]interface{}))
			}
		}

		// If neither "rules" or "rule_group" are set at this point, neither have changed but
		// at least one must still be sent to allow other attributes (ex. description) to update.
		// Give precedence again to "rules", as documented above.
		if input.Rules == nil && input.RuleGroup == nil {
			if v, ok := d.GetOk("rules"); ok {
				input.Rules = aws.String(v.(string))
			} else if v, ok := d.GetOk("rule_group"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
				input.RuleGroup = expandRuleGroup(v.([]interface{})[0].(map[string]interface{}))
			}
		}

		_, err := conn.UpdateRuleGroupWithContext(ctx, input)

		if err != nil {
			return diag.Errorf("updating NetworkFirewall Rule Group (%s): %s", d.Id(), err)
		}
	}

	return resourceRuleGroupRead(ctx, d, meta)
}

func resourceRuleGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	const (
		timeout = 10 * time.Minute
	)
	conn := meta.(*conns.AWSClient).NetworkFirewallConn()

	log.Printf("[DEBUG] Deleting NetworkFirewall Rule Group: %s", d.Id())
	_, err := tfresource.RetryWhenAWSErrMessageContains(ctx, timeout, func() (interface{}, error) {
		return conn.DeleteRuleGroupWithContext(ctx, &networkfirewall.DeleteRuleGroupInput{
			RuleGroupArn: aws.String(d.Id()),
		})
	}, networkfirewall.ErrCodeInvalidOperationException, "Unable to delete the object because it is still in use")

	if tfawserr.ErrCodeEquals(err, networkfirewall.ErrCodeResourceNotFoundException) {
		return nil
	}

	if err != nil {
		return diag.Errorf("deleting NetworkFirewall Rule Group (%s): %s", d.Id(), err)
	}

	if _, err := waitRuleGroupDeleted(ctx, conn, d.Id(), timeout); err != nil {
		return diag.Errorf("waiting for NetworkFirewall Rule Group (%s) delete : %s", d.Id(), err)
	}

	return nil
}

func FindRuleGroupByARN(ctx context.Context, conn *networkfirewall.NetworkFirewall, arn string) (*networkfirewall.DescribeRuleGroupOutput, error) {
	input := &networkfirewall.DescribeRuleGroupInput{
		RuleGroupArn: aws.String(arn),
	}

	output, err := conn.DescribeRuleGroupWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, networkfirewall.ErrCodeResourceNotFoundException) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.RuleGroupResponse == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}

func statusRuleGroup(ctx context.Context, conn *networkfirewall.NetworkFirewall, arn string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindRuleGroupByARN(ctx, conn, arn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output.RuleGroup, aws.StringValue(output.RuleGroupResponse.RuleGroupStatus), nil
	}
}

func waitRuleGroupDeleted(ctx context.Context, conn *networkfirewall.NetworkFirewall, arn string, timeout time.Duration) (*networkfirewall.RuleGroup, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{networkfirewall.ResourceStatusDeleting},
		Target:  []string{},
		Refresh: statusRuleGroup(ctx, conn, arn),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*networkfirewall.RuleGroup); ok {
		return output, err
	}

	return nil, err
}

func expandStatefulRuleHeader(l []interface{}) *networkfirewall.Header {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}
	header := &networkfirewall.Header{}
	if v, ok := tfMap["destination"].(string); ok && v != "" {
		header.Destination = aws.String(v)
	}
	if v, ok := tfMap["destination_port"].(string); ok && v != "" {
		header.DestinationPort = aws.String(v)
	}
	if v, ok := tfMap["direction"].(string); ok && v != "" {
		header.Direction = aws.String(v)
	}
	if v, ok := tfMap["protocol"].(string); ok && v != "" {
		header.Protocol = aws.String(v)
	}
	if v, ok := tfMap["source"].(string); ok && v != "" {
		header.Source = aws.String(v)
	}
	if v, ok := tfMap["source_port"].(string); ok && v != "" {
		header.SourcePort = aws.String(v)
	}

	return header
}

func expandStatefulRuleOptions(l []interface{}) []*networkfirewall.RuleOption {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	ruleOptions := make([]*networkfirewall.RuleOption, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		keyword := tfMap["keyword"].(string)
		option := &networkfirewall.RuleOption{
			Keyword: aws.String(keyword),
		}
		if v, ok := tfMap["settings"].(*schema.Set); ok && v.Len() > 0 {
			option.Settings = flex.ExpandStringSet(v)
		}
		ruleOptions = append(ruleOptions, option)
	}

	return ruleOptions
}

func expandRulesSourceList(l []interface{}) *networkfirewall.RulesSourceList {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}
	rulesSourceList := &networkfirewall.RulesSourceList{}
	if v, ok := tfMap["generated_rules_type"].(string); ok && v != "" {
		rulesSourceList.GeneratedRulesType = aws.String(v)
	}
	if v, ok := tfMap["target_types"].(*schema.Set); ok && v.Len() > 0 {
		rulesSourceList.TargetTypes = flex.ExpandStringSet(v)
	}
	if v, ok := tfMap["targets"].(*schema.Set); ok && v.Len() > 0 {
		rulesSourceList.Targets = flex.ExpandStringSet(v)
	}

	return rulesSourceList
}

func expandStatefulRules(l []interface{}) []*networkfirewall.StatefulRule {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	rules := make([]*networkfirewall.StatefulRule, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		rule := &networkfirewall.StatefulRule{}
		if v, ok := tfMap["action"].(string); ok && v != "" {
			rule.Action = aws.String(v)
		}
		if v, ok := tfMap["header"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
			rule.Header = expandStatefulRuleHeader(v)
		}
		if v, ok := tfMap["rule_option"].(*schema.Set); ok && v.Len() > 0 {
			rule.RuleOptions = expandStatefulRuleOptions(v.List())
		}
		rules = append(rules, rule)
	}

	return rules
}

func expandRuleGroup(tfMap map[string]interface{}) *networkfirewall.RuleGroup {
	if tfMap == nil {
		return nil
	}

	ruleGroup := &networkfirewall.RuleGroup{}
	if tfList, ok := tfMap["reference_sets"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
		referenceSets := &networkfirewall.ReferenceSets{}
		rvMap, ok := tfList[0].(map[string]interface{})
		if ok {
			if v, ok := rvMap["ip_set_references"].(*schema.Set); ok && v.Len() > 0 {
				referenceSets.IPSetReferences = expandIPSetReferences(v.List())
			}

			ruleGroup.ReferenceSets = referenceSets
		}
	}
	if tfList, ok := tfMap["rule_variables"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
		ruleVariables := &networkfirewall.RuleVariables{}
		rvMap, ok := tfList[0].(map[string]interface{})
		if ok {
			if v, ok := rvMap["ip_sets"].(*schema.Set); ok && v.Len() > 0 {
				ruleVariables.IPSets = expandIPSets(v.List())
			}
			if v, ok := rvMap["port_sets"].(*schema.Set); ok && v.Len() > 0 {
				ruleVariables.PortSets = expandPortSets(v.List())
			}
			ruleGroup.RuleVariables = ruleVariables
		}
	}
	if tfList, ok := tfMap["rules_source"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
		rulesSource := &networkfirewall.RulesSource{}
		rsMap, ok := tfList[0].(map[string]interface{})
		if ok {
			if v, ok := rsMap["rules_source_list"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
				rulesSource.RulesSourceList = expandRulesSourceList(v)
			}
			if v, ok := rsMap["rules_string"].(string); ok && v != "" {
				rulesSource.RulesString = aws.String(v)
			}
			if v, ok := rsMap["stateful_rule"].([]interface{}); ok && len(v) > 0 {
				rulesSource.StatefulRules = expandStatefulRules(v)
			}
			if v, ok := rsMap["stateless_rules_and_custom_actions"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
				rulesSource.StatelessRulesAndCustomActions = expandStatelessRulesAndCustomActions(v)
			}
			ruleGroup.RulesSource = rulesSource
		}
	}
	if tfList, ok := tfMap["stateful_rule_options"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
		statefulRuleOptions := &networkfirewall.StatefulRuleOptions{}
		sroMap, ok := tfList[0].(map[string]interface{})
		if ok {
			if v, ok := sroMap["rule_order"].(string); ok && v != "" {
				statefulRuleOptions.RuleOrder = aws.String(v)
			}
		}
		ruleGroup.StatefulRuleOptions = statefulRuleOptions
	}

	return ruleGroup
}

func expandIPSets(l []interface{}) map[string]*networkfirewall.IPSet {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := make(map[string]*networkfirewall.IPSet)
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if key, ok := tfMap["key"].(string); ok && key != "" {
			if tfList, ok := tfMap["ip_set"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
				tfMap, ok := tfList[0].(map[string]interface{})
				if ok {
					if tfSet, ok := tfMap["definition"].(*schema.Set); ok && tfSet.Len() > 0 {
						ipSet := &networkfirewall.IPSet{
							Definition: flex.ExpandStringSet(tfSet),
						}
						m[key] = ipSet
					}
				}
			}
		}
	}

	return m
}
func expandIPSetReferences(l []interface{}) map[string]*networkfirewall.IPSetReference {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := make(map[string]*networkfirewall.IPSetReference)
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if key, ok := tfMap["key"].(string); ok && key != "" {
			if tfList, ok := tfMap["ip_set_reference"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
				tfMap, ok := tfList[0].(map[string]interface{})
				if ok {
					if tfSet, ok := tfMap["reference_arn"].(string); ok && tfSet != "" {
						ipSetReference := &networkfirewall.IPSetReference{
							ReferenceArn: aws.String(tfSet),
						}
						m[key] = ipSetReference
					}
				}
			}
		}
	}

	return m
}
func expandPortSets(l []interface{}) map[string]*networkfirewall.PortSet {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := make(map[string]*networkfirewall.PortSet)
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if key, ok := tfMap["key"].(string); ok && key != "" {
			if tfList, ok := tfMap["port_set"].([]interface{}); ok && len(tfList) > 0 && tfList[0] != nil {
				tfMap, ok := tfList[0].(map[string]interface{})
				if ok {
					if tfSet, ok := tfMap["definition"].(*schema.Set); ok && tfSet.Len() > 0 {
						ipSet := &networkfirewall.PortSet{
							Definition: flex.ExpandStringSet(tfSet),
						}
						m[key] = ipSet
					}
				}
			}
		}
	}

	return m
}

func expandAddresses(l []interface{}) []*networkfirewall.Address {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	destinations := make([]*networkfirewall.Address, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		destination := &networkfirewall.Address{}
		if v, ok := tfMap["address_definition"].(string); ok && v != "" {
			destination.AddressDefinition = aws.String(v)
		}
		destinations = append(destinations, destination)
	}
	return destinations
}

func expandPortRanges(l []interface{}) []*networkfirewall.PortRange {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	ports := make([]*networkfirewall.PortRange, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		port := &networkfirewall.PortRange{}
		if v, ok := tfMap["from_port"].(int); ok {
			port.FromPort = aws.Int64(int64(v))
		}
		if v, ok := tfMap["to_port"].(int); ok {
			port.ToPort = aws.Int64(int64(v))
		}
		ports = append(ports, port)
	}
	return ports
}

func expandTCPFlags(l []interface{}) []*networkfirewall.TCPFlagField {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	tcpFlags := make([]*networkfirewall.TCPFlagField, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		tcpFlag := &networkfirewall.TCPFlagField{}
		if v, ok := tfMap["flags"].(*schema.Set); ok && v.Len() > 0 {
			tcpFlag.Flags = flex.ExpandStringSet(v)
		}
		if v, ok := tfMap["masks"].(*schema.Set); ok && v.Len() > 0 {
			tcpFlag.Masks = flex.ExpandStringSet(v)
		}
		tcpFlags = append(tcpFlags, tcpFlag)
	}
	return tcpFlags
}

func expandMatchAttributes(l []interface{}) *networkfirewall.MatchAttributes {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}
	matchAttributes := &networkfirewall.MatchAttributes{}
	if v, ok := tfMap["destination"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.Destinations = expandAddresses(v.List())
	}
	if v, ok := tfMap["destination_port"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.DestinationPorts = expandPortRanges(v.List())
	}
	if v, ok := tfMap["protocols"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.Protocols = flex.ExpandInt64Set(v)
	}
	if v, ok := tfMap["source"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.Sources = expandAddresses(v.List())
	}
	if v, ok := tfMap["source_port"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.SourcePorts = expandPortRanges(v.List())
	}
	if v, ok := tfMap["tcp_flag"].(*schema.Set); ok && v.Len() > 0 {
		matchAttributes.TCPFlags = expandTCPFlags(v.List())
	}

	return matchAttributes
}

func expandRuleDefinition(l []interface{}) *networkfirewall.RuleDefinition {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}
	rd := &networkfirewall.RuleDefinition{}
	if v, ok := tfMap["actions"].(*schema.Set); ok && v.Len() > 0 {
		rd.Actions = flex.ExpandStringSet(v)
	}
	if v, ok := tfMap["match_attributes"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
		rd.MatchAttributes = expandMatchAttributes(v)
	}
	return rd
}

func expandStatelessRules(l []interface{}) []*networkfirewall.StatelessRule {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	statelessRules := make([]*networkfirewall.StatelessRule, 0, len(l))
	for _, tfMapRaw := range l {
		tfMap, ok := tfMapRaw.(map[string]interface{})
		if !ok {
			continue
		}
		statelessRule := &networkfirewall.StatelessRule{}
		if v, ok := tfMap["priority"].(int); ok && v > 0 {
			statelessRule.Priority = aws.Int64(int64(v))
		}
		if v, ok := tfMap["rule_definition"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
			statelessRule.RuleDefinition = expandRuleDefinition(v)
		}
		statelessRules = append(statelessRules, statelessRule)
	}

	return statelessRules
}

func expandStatelessRulesAndCustomActions(l []interface{}) *networkfirewall.StatelessRulesAndCustomActions {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	s := &networkfirewall.StatelessRulesAndCustomActions{}
	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}
	if v, ok := tfMap["custom_action"].(*schema.Set); ok && v.Len() > 0 {
		s.CustomActions = expandCustomActions(v.List())
	}
	if v, ok := tfMap["stateless_rule"].(*schema.Set); ok && v.Len() > 0 {
		s.StatelessRules = expandStatelessRules(v.List())
	}

	return s
}

func flattenRuleGroup(r *networkfirewall.RuleGroup) []interface{} {
	if r == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"reference_sets":        flattenReferenceSets(r.ReferenceSets),
		"rule_variables":        flattenRuleVariables(r.RuleVariables),
		"rules_source":          flattenRulesSource(r.RulesSource),
		"stateful_rule_options": flattenStatefulRulesOptions(r.StatefulRuleOptions),
	}

	return []interface{}{m}
}

func flattenReferenceSets(rv *networkfirewall.ReferenceSets) []interface{} {
	if rv == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"ip_set_references": flattenIPSetReferences(rv.IPSetReferences),
	}

	return []interface{}{m}
}

func flattenIPSetReferences(m map[string]*networkfirewall.IPSetReference) []interface{} {
	if m == nil {
		return []interface{}{}
	}
	sets := make([]interface{}, 0, len(m))
	for k, v := range m {
		tfMap := map[string]interface{}{
			"key":              k,
			"ip_set_reference": flattenIPSetReference(v),
		}
		sets = append(sets, tfMap)
	}

	return sets
}

func flattenIPSetReference(i *networkfirewall.IPSetReference) []interface{} {
	if i == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"reference_arn": aws.StringValue(i.ReferenceArn),
	}

	return []interface{}{m}
}

func flattenRuleVariables(rv *networkfirewall.RuleVariables) []interface{} {
	if rv == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"ip_sets":   flattenIPSets(rv.IPSets),
		"port_sets": flattenPortSets(rv.PortSets),
	}

	return []interface{}{m}
}

func flattenIPSets(m map[string]*networkfirewall.IPSet) []interface{} {
	if m == nil {
		return []interface{}{}
	}
	sets := make([]interface{}, 0, len(m))
	for k, v := range m {
		tfMap := map[string]interface{}{
			"key":    k,
			"ip_set": flattenIPSet(v),
		}
		sets = append(sets, tfMap)
	}

	return sets
}

func flattenPortSets(m map[string]*networkfirewall.PortSet) []interface{} {
	if m == nil {
		return []interface{}{}
	}
	sets := make([]interface{}, 0, len(m))
	for k, v := range m {
		tfMap := map[string]interface{}{
			"key":      k,
			"port_set": flattenPortSet(v),
		}
		sets = append(sets, tfMap)
	}

	return sets
}

func flattenIPSet(i *networkfirewall.IPSet) []interface{} {
	if i == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"definition": flex.FlattenStringSet(i.Definition),
	}

	return []interface{}{m}
}

func flattenPortSet(p *networkfirewall.PortSet) []interface{} {
	if p == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"definition": flex.FlattenStringSet(p.Definition),
	}

	return []interface{}{m}
}

func flattenRulesSource(rs *networkfirewall.RulesSource) []interface{} {
	if rs == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"rules_source_list":                  flattenRulesSourceList(rs.RulesSourceList),
		"rules_string":                       aws.StringValue(rs.RulesString),
		"stateful_rule":                      flattenStatefulRules(rs.StatefulRules),
		"stateless_rules_and_custom_actions": flattenStatelessRulesAndCustomActions(rs.StatelessRulesAndCustomActions),
	}

	return []interface{}{m}
}

func flattenRulesSourceList(r *networkfirewall.RulesSourceList) []interface{} {
	if r == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"generated_rules_type": aws.StringValue(r.GeneratedRulesType),
		"target_types":         flex.FlattenStringSet(r.TargetTypes),
		"targets":              flex.FlattenStringSet(r.Targets),
	}

	return []interface{}{m}
}

func flattenStatefulRules(sr []*networkfirewall.StatefulRule) []interface{} {
	if sr == nil {
		return []interface{}{}
	}
	rules := make([]interface{}, 0, len(sr))
	for _, s := range sr {
		m := map[string]interface{}{
			"action":      aws.StringValue(s.Action),
			"header":      flattenHeader(s.Header),
			"rule_option": flattenRuleOptions(s.RuleOptions),
		}
		rules = append(rules, m)
	}
	return rules
}

func flattenHeader(h *networkfirewall.Header) []interface{} {
	if h == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"destination":      aws.StringValue(h.Destination),
		"destination_port": aws.StringValue(h.DestinationPort),
		"direction":        aws.StringValue(h.Direction),
		"protocol":         aws.StringValue(h.Protocol),
		"source":           aws.StringValue(h.Source),
		"source_port":      aws.StringValue(h.SourcePort),
	}

	return []interface{}{m}
}

func flattenRuleOptions(o []*networkfirewall.RuleOption) []interface{} {
	if o == nil {
		return []interface{}{}
	}

	options := make([]interface{}, 0, len(o))
	for _, option := range o {
		m := map[string]interface{}{
			"keyword":  aws.StringValue(option.Keyword),
			"settings": aws.StringValueSlice(option.Settings),
		}
		options = append(options, m)
	}

	return options
}

func flattenStatelessRulesAndCustomActions(sr *networkfirewall.StatelessRulesAndCustomActions) []interface{} {
	if sr == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"custom_action":  flattenCustomActions(sr.CustomActions),
		"stateless_rule": flattenStatelessRules(sr.StatelessRules),
	}

	return []interface{}{m}
}

func flattenStatelessRules(sr []*networkfirewall.StatelessRule) []interface{} {
	if sr == nil {
		return []interface{}{}
	}

	rules := make([]interface{}, 0, len(sr))
	for _, s := range sr {
		rule := map[string]interface{}{
			"priority":        int(aws.Int64Value(s.Priority)),
			"rule_definition": flattenRuleDefinition(s.RuleDefinition),
		}
		rules = append(rules, rule)
	}

	return rules
}

func flattenRuleDefinition(r *networkfirewall.RuleDefinition) []interface{} {
	if r == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"actions":          flex.FlattenStringSet(r.Actions),
		"match_attributes": flattenMatchAttributes(r.MatchAttributes),
	}

	return []interface{}{m}
}

func flattenMatchAttributes(ma *networkfirewall.MatchAttributes) []interface{} {
	if ma == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"destination":      flattenAddresses(ma.Destinations),
		"destination_port": flattenPortRanges(ma.DestinationPorts),
		"protocols":        flex.FlattenInt64Set(ma.Protocols),
		"source":           flattenAddresses(ma.Sources),
		"source_port":      flattenPortRanges(ma.SourcePorts),
		"tcp_flag":         flattenTCPFlags(ma.TCPFlags),
	}

	return []interface{}{m}
}

func flattenAddresses(d []*networkfirewall.Address) []interface{} {
	if d == nil {
		return []interface{}{}
	}

	destinations := make([]interface{}, 0, len(d))
	for _, addr := range d {
		m := map[string]interface{}{
			"address_definition": aws.StringValue(addr.AddressDefinition),
		}
		destinations = append(destinations, m)
	}

	return destinations
}

func flattenPortRanges(pr []*networkfirewall.PortRange) []interface{} {
	if pr == nil {
		return []interface{}{}
	}

	portRanges := make([]interface{}, 0, len(pr))
	for _, r := range pr {
		m := map[string]interface{}{
			"from_port": int(aws.Int64Value(r.FromPort)),
			"to_port":   int(aws.Int64Value(r.ToPort)),
		}
		portRanges = append(portRanges, m)
	}

	return portRanges
}

func flattenTCPFlags(t []*networkfirewall.TCPFlagField) []interface{} {
	if t == nil {
		return []interface{}{}
	}
	flagFields := make([]interface{}, 0, len(t))
	for _, v := range t {
		m := map[string]interface{}{
			"flags": flex.FlattenStringSet(v.Flags),
			"masks": flex.FlattenStringSet(v.Masks),
		}
		flagFields = append(flagFields, m)
	}

	return flagFields
}

func flattenStatefulRulesOptions(sro *networkfirewall.StatefulRuleOptions) []interface{} {
	if sro == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"rule_order": aws.StringValue(sro.RuleOrder),
	}

	return []interface{}{m}
}
