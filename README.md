# Terraform Provider for Google Ads

`terraform-provider-googleads` is a Terraform Plugin Framework provider for managing practical Google Ads Search campaign structures as code.

This first version uses the Google Ads REST API directly instead of the official generated Go client. The Google Ads API is exposed as stable REST endpoints, and the provider only needs OAuth2 plus a small set of mutate/search operations. Direct REST keeps the initial provider small, readable, and easy to audit while still using official Google Ads API resource names, services, GAQL reads, and mutate semantics.

## Supported resources

- `googleads_campaign_budget`
- `googleads_campaign`
- `googleads_ad_group`
- `googleads_ad_group_keyword`
- `googleads_ad_group_ad` for responsive search ads
- `googleads_campaign_criterion` for location and language criteria

Each resource stores the Google Ads `resource_name` as the stable Terraform `id` and exposes it as `resource_name` for references.

## Provider configuration

Arguments can be set directly in Terraform or through environment variables:

- `developer_token` / `GOOGLEADS_DEVELOPER_TOKEN` *(sensitive)*
- `customer_id` / `GOOGLEADS_CUSTOMER_ID`
- `login_customer_id` / `GOOGLEADS_LOGIN_CUSTOMER_ID`
- `client_id` / `GOOGLEADS_CLIENT_ID` *(sensitive)*
- `client_secret` / `GOOGLEADS_CLIENT_SECRET` *(sensitive)*
- `refresh_token` / `GOOGLEADS_REFRESH_TOKEN` *(sensitive)*
- `token_file` / `GOOGLEADS_TOKEN_FILE`
- `credentials_file` / `GOOGLEADS_CREDENTIALS_FILE`
- `api_version` / `GOOGLEADS_API_VERSION` defaults to `v23`
- `validate_only` / `GOOGLEADS_VALIDATE_ONLY`

You can authenticate either with:

1. `credentials_file` plus `token_file`; or
2. `credentials_file` plus `refresh_token`; or
3. `client_id`, `client_secret`, and `refresh_token`.

Secrets are marked sensitive in the Terraform schema and are not logged by the provider.

## Normalization rules

The provider normalizes equivalent Google Ads inputs to avoid unnecessary Terraform diffs:

- `customer_id` and `login_customer_id` may be set with or without dashes. They are sent to Google Ads without dashes.
- Enum-like fields such as `status`, `delivery_method`, `advertising_channel_type`, `bidding_strategy_type`, `type`, and `match_type` are normalized to uppercase snake case. For example, `manual-cpc` becomes `MANUAL_CPC`.
- Google Ads resource references may be full resource names such as `customers/1234567890/campaigns/111`. If a full resource name contains a dashed customer ID, the customer segment is normalized.
- For customer-scoped references such as `campaign_budget`, `campaign`, and `ad_group`, numeric IDs and collection-relative IDs are accepted in CRUD calls and expanded with the configured `customer_id` where safe. Examples: `111` or `campaigns/111` becomes `customers/<customer_id>/campaigns/111`.
- For campaign criterion constants, numeric IDs are accepted: `2840` becomes `geoTargetConstants/2840`, and `1000` becomes `languageConstants/1000`.
- Imported resources still use full Google Ads resource names. After a resource has state, numeric or collection-relative references that point at the same remote object are planned as the canonical resource name to suppress equivalent diffs.

## Validation rules

Terraform validates common mistakes before calling the Google Ads API:

- Enum-like fields reject unsupported values with a message listing the accepted values. Hyphenated/lowercase input is accepted when it normalizes to a supported Google Ads enum value.
- Campaign `start_date` and `end_date` must use `YYYY-MM-DD`.
- `bidding_strategy_type = "TARGET_CPA"` requires positive `target_cpa_micros`.
- `bidding_strategy_type = "TARGET_ROAS"` requires positive `target_roas`.
- `googleads_campaign_criterion.type = "LOCATION"` requires `location_geo_target_constant`.
- `googleads_campaign_criterion.type = "LANGUAGE"` requires `language_constant`.
- Responsive search ads require 3-15 headlines and 2-4 descriptions. Headlines must be 30 characters or fewer; descriptions must be 90 characters or fewer.

## Google Ads API access

1. Create or choose a Google Cloud project.
2. Enable the Google Ads API for that project.
3. Configure an OAuth consent screen.
4. Create OAuth client credentials. A Desktop app client is easiest for the local helper flow.
5. Download the OAuth client credentials JSON as `credentials.json`.
6. In Google Ads, request or locate your developer token under **Tools & Settings → Setup → API Center**.
7. Use a Google Ads customer ID without dashes, for example `1234567890`. If you use a manager account, set `login_customer_id` to the manager account ID.

## Generate a refresh token

Build and run the helper CLI:

```bash
go build -o bin/googleads-auth ./cmd/googleads-auth
./bin/googleads-auth -credentials ./credentials.json -out ./token.json
```

The helper prints a browser URL, asks you to paste the authorization code, saves `token.json`, and prints provider environment variables. The saved token JSON contains a refresh token, so keep it private.

## Install local provider

```bash
make install-local VERSION=0.1.0 OS_ARCH=linux_amd64
```

This installs the provider at:

```text
~/.terraform.d/plugins/registry.terraform.io/local/googleads/0.1.0/linux_amd64/terraform-provider-googleads_v0.1.0
```

Adjust `OS_ARCH` for your platform if needed, for example `darwin_arm64`.

## Example Terraform

```hcl
terraform {
  required_providers {
    googleads = {
      source  = "local/googleads"
      version = "0.1.0"
    }
  }
}

provider "googleads" {
  developer_token   = var.googleads_developer_token
  customer_id       = var.googleads_customer_id
  login_customer_id = var.googleads_login_customer_id

  credentials_file = "./credentials.json"
  token_file       = "./token.json"

  validate_only = false
}

resource "googleads_campaign_budget" "search_budget" {
  name          = "terraform-search-budget"
  amount_micros = 5000000
}

resource "googleads_campaign" "search" {
  name                     = "terraform-search-campaign"
  advertising_channel_type = "SEARCH"
  status                   = "PAUSED"
  campaign_budget          = googleads_campaign_budget.search_budget.resource_name

  start_date = "2026-07-01"
  end_date   = "2026-12-31"

  bidding_strategy_type = "MANUAL_CPC"

  network_settings = {
    target_google_search          = true
    target_search_network         = true
    target_content_network        = false
    target_partner_search_network = false
  }
}

resource "googleads_campaign_criterion" "us" {
  campaign                     = googleads_campaign.search.resource_name
  type                         = "LOCATION"
  location_geo_target_constant = "geoTargetConstants/2840"
}

resource "googleads_campaign_criterion" "english" {
  campaign          = googleads_campaign.search.resource_name
  type              = "LANGUAGE"
  language_constant = "languageConstants/1000"
}

resource "googleads_ad_group" "main" {
  campaign       = googleads_campaign.search.resource_name
  name           = "terraform-main-ad-group"
  status         = "PAUSED"
  type           = "SEARCH_STANDARD"
  cpc_bid_micros = 1000000
}

resource "googleads_ad_group_keyword" "keyword" {
  ad_group   = googleads_ad_group.main.resource_name
  text       = "terraform devops automation"
  match_type = "PHRASE"
  status     = "ENABLED"
}

resource "googleads_ad_group_ad" "rsa" {
  ad_group = googleads_ad_group.main.resource_name
  status   = "PAUSED"

  final_urls = ["https://example.com"]

  path1 = "devops"
  path2 = "automation"

  responsive_search_ad = {
    headlines = [
      "Automate DevOps Workflows",
      "Terraform Managed Campaigns",
      "Reliable Infra Automation",
    ]

    descriptions = [
      "Create and manage your campaigns as code.",
      "Use Terraform workflows for repeatable Google Ads setup.",
    ]
  }
}
```

A copy of this example is in `examples/search_campaign`.

## Run Terraform

```bash
cd examples/search_campaign
terraform init
terraform plan
terraform apply
terraform destroy
```

For a dry API validation without creating resources, set:

```bash
export GOOGLEADS_VALIDATE_ONLY=true
```

or set `validate_only = true` in the provider block.

## API diagnostics

When Google Ads API calls fail, the provider parses `GoogleAdsFailure` details from REST responses and surfaces actionable Terraform diagnostics including the Google Ads error code, message, trigger, field path, and operation index when available. Request credentials and token values are not included in diagnostics.

## Retry behavior

Google Ads API requests are retried conservatively for transient failures: HTTP `429`, HTTP `5xx`, and Google Ads statuses `RESOURCE_EXHAUSTED`, `UNAVAILABLE`, and `DEADLINE_EXCEEDED` when identifiable from an error response. The provider makes up to 3 attempts total with exponential backoff, jitter, and a maximum delay of 2 seconds. Validation and business-rule errors such as `INVALID_ARGUMENT` are not retried.

## Delete behavior

Google Ads resources are generally removed through each service's `remove` operation rather than physically deleted from historical reporting. The provider uses remove operations for campaign budgets, campaigns, ad groups, ad group criteria, ad group ads, and campaign criteria. Removed resources may remain visible in Google Ads history/reporting, but Terraform removes them from state after a successful destroy.

## Imports

Resources can be imported by full Google Ads resource name. After importing, run `terraform plan` and update your configuration to match any remote-only values Terraform reads back from Google Ads.

```bash
terraform import googleads_campaign_budget.search_budget customers/1234567890/campaignBudgets/222
terraform import googleads_campaign.search customers/1234567890/campaigns/111
terraform import googleads_ad_group.main customers/1234567890/adGroups/333
terraform import googleads_ad_group_keyword.keyword customers/1234567890/adGroupCriteria/333~444
terraform import googleads_ad_group_ad.rsa customers/1234567890/adGroupAds/333~555
terraform import googleads_campaign_criterion.us customers/1234567890/campaignCriteria/111~2840
```

Import notes:

- `googleads_ad_group_ad` reads back required RSA fields including `final_urls`, `path1`, `path2`, `headlines`, and `descriptions` so imported ads can be represented in Terraform configuration.
- `googleads_campaign_criterion` supports imported location and language criteria in this first version.
- Google Ads resource names are account-scoped; import with the same `customer_id` and optional `login_customer_id` used by the provider.

## Development

```bash
make fmt
make test
make build
```

Acceptance tests are structured but disabled by default because they require real Google Ads credentials and can mutate an account:

```bash
TF_ACC=1 \
GOOGLEADS_DEVELOPER_TOKEN=... \
GOOGLEADS_CUSTOMER_ID=... \
GOOGLEADS_CREDENTIALS_FILE=./credentials.json \
GOOGLEADS_TOKEN_FILE=./token.json \
go test ./internal/provider -run TestAcc
```

## Notes and current limitations

- The initial provider focuses on Search campaign setup.
- Responsive search ads are the first supported ad type.
- Conversion actions are not implemented in this first pass; add them as a follow-up `googleads_conversion_action` resource using `ConversionActionService`.
- GAQL reads are used through `GoogleAdsService.Search`; writes use resource-specific mutate endpoints.
- Partial failures are disabled so Terraform receives clear all-or-nothing API failures.
