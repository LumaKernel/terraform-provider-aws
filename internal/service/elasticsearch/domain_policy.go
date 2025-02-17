package elasticsearch

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceDomainPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceDomainPolicyUpsert,
		Read:   resourceDomainPolicyRead,
		Update: resourceDomainPolicyUpsert,
		Delete: resourceDomainPolicyDelete,

		Schema: map[string]*schema.Schema{
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"access_policies": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
			},
		},
	}
}

func resourceDomainPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).ElasticSearchConn
	name := d.Get("domain_name").(string)
	out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] ElasticSearch Domain %q not found, removing", name)
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received ElasticSearch domain: %s", out)

	ds := out.DomainStatus
	d.Set("access_policies", ds.AccessPolicies)

	return nil
}

func resourceDomainPolicyUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).ElasticSearchConn
	domainName := d.Get("domain_name").(string)
	_, err := conn.UpdateElasticsearchDomainConfig(&elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName:     aws.String(domainName),
		AccessPolicies: aws.String(d.Get("access_policies").(string)),
	})
	if err != nil {
		return err
	}

	d.SetId("esd-policy-" + domainName)
	input := &elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	}
	var out *elasticsearch.DescribeElasticsearchDomainOutput
	err = resource.Retry(50*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.DescribeElasticsearchDomain(input)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if !*out.DomainStatus.Processing {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for changes to be processed", d.Id()))
	})
	if tfresource.TimedOut(err) {
		out, err = conn.DescribeElasticsearchDomain(input)
		if err == nil && !*out.DomainStatus.Processing {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error upserting Elasticsearch domain policy: %s", err)
	}

	return resourceDomainPolicyRead(d, meta)
}

func resourceDomainPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).ElasticSearchConn

	_, err := conn.UpdateElasticsearchDomainConfig(&elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName:     aws.String(d.Get("domain_name").(string)),
		AccessPolicies: aws.String(""),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for ElasticSearch domain policy %q to be deleted", d.Get("domain_name").(string))
	input := &elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	}
	var out *elasticsearch.DescribeElasticsearchDomainOutput
	err = resource.Retry(60*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.DescribeElasticsearchDomain(input)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if !*out.DomainStatus.Processing {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for policy to be deleted", d.Id()))
	})
	if tfresource.TimedOut(err) {
		out, err := conn.DescribeElasticsearchDomain(input)
		if err == nil && !*out.DomainStatus.Processing {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting Elasticsearch domain policy: %s", err)
	}
	return nil
}
