package cloudwatchevents_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccCloudWatchEventsSourceDataSource_basic(t *testing.T) {
	key := "EVENT_BRIDGE_PARTNER_EVENT_SOURCE_NAME"
	busName := os.Getenv(key)
	if busName == "" {
		t.Skipf("Environment variable %s is not set", key)
	}

	parts := strings.Split(busName, "/")
	if len(parts) < 2 {
		t.Errorf("unable to parse partner event bus name %s", busName)
	}
	createdBy := parts[0] + "/" + parts[1]

	dataSourceName := "data.aws_cloudwatch_event_source.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, cloudwatchevents.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccPartnerEventSourceDataSourceConfig(busName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", busName),
					resource.TestCheckResourceAttr(dataSourceName, "created_by", createdBy),
					resource.TestCheckResourceAttrSet(dataSourceName, "arn"),
				),
			},
		},
	})
}

func testAccPartnerEventSourceDataSourceConfig(namePrefix string) string {
	return fmt.Sprintf(`
data "aws_cloudwatch_event_source" "test" {
  name_prefix = "%s"
}
`, namePrefix)
}
