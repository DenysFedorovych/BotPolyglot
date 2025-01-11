variable "bot_secret" {
  description = "Secret token for Telegram bot."
  type        = string
  default     = "REPLACE_WITH_YOUR_SECRET"
}

variable "bot_token" {
  description = "Token for Telegram bot."
  type        = string
  default     = "REPLACE_WITH_YOUR_TOKEN"
}

variable "deepl_auth_key" {
  description = "DeepL API authentication key."
  type        = string
  default     = "REPLACE_WITH_YOUR_KEY"
}

variable "target_lang" {
  description = "Target language for translation."
  type        = string
  default     = "EN"
}

variable "unique_s3_bucket_name" {
  description = "Unique S3 bucket name that will be created on your account."
  type        = string
  default     = "REPLACE_IT_WITH_YOUR"
}
