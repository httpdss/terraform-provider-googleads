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
    target_google_search           = true
    target_search_network          = true
    target_content_network         = false
    target_partner_search_network  = false
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
    # Google Ads responsive search ads require 3-15 headlines (max 30 characters each)
    # and 2-4 descriptions (max 90 characters each).
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
