# Copyright 2024 Hewlett Packard Enterprise Development LP

variable "name" {
    type        = string
    description = "Your name."
    default = "world"
}

variable "age" {
    type        = number
    description = "Your age. Required."
}