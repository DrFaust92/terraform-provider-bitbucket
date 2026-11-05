## Unreleased

### Bug fixes

* `bitbucket_branch_restriction`: `users` and `groups` previously failed to populate in state at all, because the Bitbucket API's GDPR changes removed usernames from account responses while the provider still expected them. Now: `users` round-trips via UUID, and `groups.owner` round-trips via workspace slug or UUID (whichever form is already in your config/state). **Behavior change:** a config that lists a username in `users` previously showed no diff (because `Read` silently failed to populate state); it will now show a diff on every `plan`/`apply` until you switch that entry to the UUID shown in the new warning, or add `lifecycle { ignore_changes = [users] }`.

## 1.3.0 (March 15, 2021)

This release contains the changes to upstream repo that were never released.

### Features

* add `bitbucket_deployment` and `bitbucket_deployment_variable` resources [#60](https://github.com/hashicorp/terraform-provider-bitbucket/pull/60)
* add `require_default_reviewer_approvals_to_merge` branch restriction value [#52](https://github.com/hashicorp/terraform-provider-bitbucket/pull/52)

### Bug fixes

* fix issue with omitempty [#49](https://github.com/hashicorp/terraform-provider-bitbucket/pull/49)

### Documentation

* fix ducmentation typo [#54](https://github.com/hashicorp/terraform-provider-bitbucket/pull/54), [#61](https://github.com/hashicorp/terraform-provider-bitbucket/pull/61) and [#65](https://github.com/hashicorp/terraform-provider-bitbucket/pull/65)

## 1.2.0 (January 23, 2020)
* add `bitbucket_project` to create a new project via the API
* add `bitbucket_repository` turn on/off pipelines
* add `bitbucket_repository_variable` to add variables via terraform to your pipelines builds
* add `bitbucket_user` to find a user and use for default reviewers.

## 1.1.0 (June 19, 2019)

### Features

* add `skip_cert_verification` for hooks [#19](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/19)

### Bug fixes

* handle missing hooks [#24](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/24)
* fix default reviewer pagination bug [#28](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/28)

### Dev updates

* add `website` and `website-test` targets [#16](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/16)
* add `website-test` target to Travis [#17](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/17)
* upgrade to go 1.11 [#25](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/25)
* switch to go modules [#27](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/27)
* upgrade to `hashicorp/terraform` v0.12.2 [#34](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/34)

### Documentation

* add note about v1 APIs [#21](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/21)

## 1.0.0 (December 08, 2017)

* resource/bitbucket_repository: Add the ability to define a seperate slug for a repository ([#5](https://github.com/terraform-providers/terraform-provider-bitbucket/issues/5))

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
