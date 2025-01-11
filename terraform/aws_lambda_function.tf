resource "aws_lambda_function" "tg_message_webhook" {
  function_name = "tg-message-webhook"
  role          = aws_iam_role.lambda_exec_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  memory_size   = 128
  timeout       = 3
  architectures = ["x86_64"]

  # Referencing the S3 object for the Lambda function package
  s3_bucket        = aws_s3_bucket.lambda_bucket.id
  s3_key           = aws_s3_object.lambda_zip.key
  source_code_hash = filebase64sha256("function.zip") # Ensures updates deploy correctly

  environment {
    variables = {
      BOT_SECRET     = var.bot_secret
      BOT_TOKEN      = var.bot_token
      DEEPL_AUTH_KEY = var.deepl_auth_key
      TARGET_LANG    = var.target_lang
    }
  }
}
