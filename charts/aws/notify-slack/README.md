# notify-slack

Port of [terraform-aws-modules/terraform-aws-notify-slack](https://github.com/terraform-aws-modules/terraform-aws-notify-slack/tree/v7.5.2) (`v7.5.2`) to AWS Controllers for Kubernetes (ACK) objects.

## What it renders

- `cloudwatchlogs.services.k8s.aws/v1alpha1` `LogGroup` (`/aws/lambda/<lambda_function_name>`)
- `sns.services.k8s.aws/v1alpha1` `Topic` (when `create_sns_topic` is true) and `Subscription`
- `iam.services.k8s.aws/v1alpha1` `Role` for the Lambda function (when `lambda_role` is empty) and optional `Role` for SNS delivery status feedback
- `lambda.services.k8s.aws/v1alpha1` `Function` (configured with `s3_existing_package` or `image_uri`)
- `lambda.aws.hfl.dev/v1alpha1` `Permission` (when `allow_alpha: true`)

## Differences from Terraform

- Local zip packaging (`lambda_source_path`, `hash_extra`, `recreate_missing_package`, `lambda_function_store_on_s3`, `trigger_on_package_timestamp`) is not portable; supply a published artifact via `s3_existing_package` (`{bucket, key, version_id}`) or `image_uri`.
- Plaintext `slack_webhook_url` is refused when `kms_key_arn` is empty (`docs/secrets.md`); pass a KMS-encrypted ciphertext with `kms_key_arn` set.
- `account_id` and `region` are required to build the Lambda endpoint ARN (and topic ARN when `create_sns_topic` is false) for the SNS subscription.
