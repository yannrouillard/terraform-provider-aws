package kinesisanalyticsv2_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/kinesisanalyticsv2"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfkinesisanalyticsv2 "github.com/hashicorp/terraform-provider-aws/internal/service/kinesisanalyticsv2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccKinesisAnalyticsV2ApplicationSnapshot_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var v kinesisanalyticsv2.SnapshotDetails
	resourceName := "aws_kinesisanalyticsv2_application_snapshot.test"
	applicationResourceName := "aws_kinesisanalyticsv2_application.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, kinesisanalyticsv2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckApplicationSnapshotDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationSnapshotConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApplicationSnapshotExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "application_name", applicationResourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "application_version_id", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "snapshot_creation_timestamp"),
					resource.TestCheckResourceAttr(resourceName, "snapshot_name", rName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKinesisAnalyticsV2ApplicationSnapshot_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var v kinesisanalyticsv2.SnapshotDetails
	resourceName := "aws_kinesisanalyticsv2_application_snapshot.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, kinesisanalyticsv2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckApplicationSnapshotDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationSnapshotConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApplicationSnapshotExists(ctx, resourceName, &v),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfkinesisanalyticsv2.ResourceApplicationSnapshot(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccKinesisAnalyticsV2ApplicationSnapshot_Disappears_application(t *testing.T) {
	ctx := acctest.Context(t)
	var v kinesisanalyticsv2.SnapshotDetails
	resourceName := "aws_kinesisanalyticsv2_application_snapshot.test"
	applicationResourceName := "aws_kinesisanalyticsv2_application.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, kinesisanalyticsv2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckApplicationSnapshotDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationSnapshotConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApplicationSnapshotExists(ctx, resourceName, &v),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfkinesisanalyticsv2.ResourceApplication(), applicationResourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckApplicationSnapshotDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).KinesisAnalyticsV2Conn()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_kinesisanalyticsv2_application_snapshot" {
				continue
			}

			_, err := tfkinesisanalyticsv2.FindSnapshotDetailsByApplicationAndSnapshotNames(ctx, conn, rs.Primary.Attributes["application_name"], rs.Primary.Attributes["snapshot_name"])

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Kinesis Analytics v2 Application Snapshot %s still exists", rs.Primary.ID)
		}
		return nil
	}
}

func testAccCheckApplicationSnapshotExists(ctx context.Context, n string, v *kinesisanalyticsv2.SnapshotDetails) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Kinesis Analytics v2 Application Snapshot ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).KinesisAnalyticsV2Conn()

		application, err := tfkinesisanalyticsv2.FindSnapshotDetailsByApplicationAndSnapshotNames(ctx, conn, rs.Primary.Attributes["application_name"], rs.Primary.Attributes["snapshot_name"])

		if err != nil {
			return err
		}

		*v = *application

		return nil
	}
}

func testAccApplicationSnapshotConfig_basic(rName string) string {
	return testAccApplicationConfig_startSnapshotableFlink(rName, "SKIP_RESTORE_FROM_SNAPSHOT", "", false)
}
