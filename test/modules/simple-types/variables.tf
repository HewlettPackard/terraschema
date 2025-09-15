# Copyright 2024 Hewlett Packard Enterprise Development LP

variable "a_string" {
  type        = string
  default     = "a string"
  description = "This is a string"
}

variable "a_number" {
  type        = number
  description = "This is a number"
}

variable "a_bool" {
  type        = bool
  default     = false
  description = "This is a boolean"
}

variable "a_nullable_string" {
  type        = string
  nullable    = true
  description = "This is a nullable string"
}

variable "a_list" {
  type        = list(string)
  default     = ["a", "b", "c"]
  description = "This is a list of strings"
}

variable "a_map_of_strings" {
  type        = map(string)
  default     = {
    a = "a"
    b = "b"
    c = "c"
  }
  description = "This is a map of strings"
}

variable "an_object" {
  type        = object({
    a = string
    b = number
    c = bool
  })
  default     = {
    a = "a"
    b = 1
    c = true
  }
  description = "This is an object"
}

variable "a_tuple" {
  type        = tuple([string, number, bool])
  default     = ["a", 1, true]
  description = "This is a tuple"
}

variable "a_set" {
  type        = set(string)
  default     = ["a", "b", "c"]
  description = "This is a set of strings"
}

variable "an_any_as_map" {
  type        = any
  default     = {}
  description = "This is an any"
}

variable "an_any_as_list" {
  type        = any
  default     = []
  description = "This is an any"
}

variable "an_any_as_string" {
  type        = any
  default     = "default"
  description = "This is an any"
}

variable "an_any_as_number" {
  type        = any
  default     = 1
  description = "This is an any"
}

variable "an_any_as_boolean" {
  type        = any
  default     = true
  description = "This is an any"
}

variable "a_list_of_any" {
  type        = list(any)
  default     = ["a", "b", "c"]
  description = "This is a list of any"
}

variable "a_map_of_any" {
  type    = map(any)
  default = {
    a = "a"
    b = "b"
    c = "c"
  }
  description = "This is a map of any"
}

variable "a_set_of_any" {
  type        = set(any)
  default     = ["a", "b", "c"]
  description = "This is a set of any"
}

variable "an_unspecified_as_map" {
  default     = {}
  description = "This is an unspecified"
}

variable "an_unspecified_as_list" {
  default     = []
  description = "This is an unspecified"
}

variable "an_unspecified_as_string" {
  default     = "default"
  description = "This is an unspecified"
}

variable "an_unspecified_as_number" {
  default     = 1
  description = "This is an unspecified"
}

variable "an_unspecified_as_boolean" {
  default     = true
  description = "This is an unspecified"
}
