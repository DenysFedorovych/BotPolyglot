resource "aws_s3_bucket" "lambda_bucket" {
  bucket = var.unique_s3_bucket_name # Replace with your desired unique bucket name
}

resource "aws_s3_object" "lambda_zip" {
  bucket = aws_s3_bucket.lambda_bucket.id
  key    = "function.zip"
  source = "function.zip" # Path to your local zip file
  etag   = filemd5("function.zip")

  tags = {
    Name = "LambdaFunctionCode"
  }
}
