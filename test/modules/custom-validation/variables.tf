# Copyright 2024 Hewlett Packard Enterprise Development LP

variable "a_string_enum_kind_1" {
  type        = string
  default     = "a"
  description = "A string variable that must be one of the values 'a', 'b', or 'c'"
  validation {
    condition      = contains(["a", "b", "c"], var.a_string_enum_kind_1)
    error_message  = "Invalid value for a_string_enum_kind_1"
  }
}

variable "a_string_enum_kind_2" {
  type        = string
  default     = "a"
  description = "A string variable that must be one of the values 'a', 'b', or 'c'"
  validation {
    condition      = var.a_string_enum_kind_2 == "a" || var.a_string_enum_kind_2 == "b" || var.a_string_enum_kind_2 == "c"
    error_message  = "Invalid value for a_string_enum_kind_2"
  }
}

variable "a_number_enum_kind_1" {
  type        = number
  default     = 1
  description = "A number variable that must be one of the values 1, 2, or 3"
  validation {
    condition      = contains([1, 2, 3], var.a_number_enum_kind_1)
    error_message  = "Invalid value for a_number_enum_kind_1"
  }
}

variable "a_number_enum_kind_2" {
  type        = number
  default     = 1
  description = "A number variable that must be one of the values 1, 2, or 3"
  validation {
    condition      = var.a_number_enum_kind_2 == 1 || var.a_number_enum_kind_2 == 2 || var.a_number_enum_kind_2 == 3
    error_message  = "Invalid value for a_number_enum_kind_2"
  }
}

variable "a_number_exclusive_maximum_minimum" {
  type        = number
  default = 1
  description = "A number variable that must be greater than 0 and less than 10"
  validation {
    condition      = var.a_number_exclusive_maximum_minimum > 0 && var.a_number_exclusive_maximum_minimum < 10
    error_message  = "a_number_exclusive_maximum_minimum must be less than 10 and greater than 0"
  }
}


variable "a_number_maximum_minimum" {
  type        = number
  default = 0
  description = "A number variable that must be between 0 and 10 (inclusive)"
  validation {
    condition      = var.a_number_maximum_minimum >= 0 && var.a_number_maximum_minimum <= 10
    error_message  = "a_number_maximum_minimum must be less than or equal to 10 and greater than or equal to 0"
  }
}


variable "a_list_maximum_minimum_length" {
  type        = list(string)
  default = [ "a" ]
  description = "A list variable that must have a length greater than 0 and less than 10"
  validation {
    condition      = length(var.a_list_maximum_minimum_length) > 0 && length(var.a_list_maximum_minimum_length) < 10
    error_message  = "a_list_maximum_minimum_length must have a length greater than 0 and less than 10"
  }
}

variable "an_object_maximum_minimum_items" {
  type        = object({
    name = string
  })
  description = "An object variable that must have fewer than 3 properties"
  validation {
    condition      = length(var.an_object_maximum_minimum_items) > 0 && length(var.an_object_maximum_minimum_items) < 3
    error_message  = "an_object_maximum_minimum_items must have fewer than 3 properties"
  }
  default = {
    name        = "a"
    other_field = "b"
  }
}

variable "a_map_maximum_minimum_entries" {
  type        = map(string)
  description = "A map variable that must have greater than 0 and less than 10 entries"
  validation {
    condition      = length(var.a_map_maximum_minimum_entries) > 0 &&  length(var.a_map_maximum_minimum_entries)< 10
    error_message  = "a_map_maximum_minimum_entries must greater than 0 and less than 10 entries"
  }
  default = {
    "a" = "a"
  }
}

variable "a_set_maximum_minimum_items" {
  type        = set(string)
  description = "A set variable that must have a length greater than 0 and less than 10"
  validation {
    condition      = 0 < length(var.a_set_maximum_minimum_items) && 10 > length(var.a_set_maximum_minimum_items)
    error_message  = "a_set_maximum_minimum_items must have a length greater than 0 and less than 10"
  }
  default = ["a"]
}

variable "a_string_maximum_minimum_length" {
  type        = string
  description = "A string variable that must have a length less than 10 and greater than 0"
  validation {
    condition      =0<length(var.a_string_maximum_minimum_length)&&length(var.a_string_maximum_minimum_length)<10
    error_message  = "a_string_maximum_minimum_length must have a length less than 10 and greater than 0"
  }
  default = "a"
}

variable "a_string_set_length" {
  type        = string
  description = "A string variable that must have length 4"
  validation {
    condition      = 4==length(var.a_string_set_length)
    error_message  = "a_string_set_length must have length 4"
  }
  default = "abcd"
}

variable "a_string_length_over_defined" {
  type        = string
  description = "A string variable that must have length 4"
  validation {
    condition      = 2<length(var.a_string_length_over_defined)&&length(var.a_string_length_over_defined) == 4&& 7>length(var.a_string_length_over_defined)
    error_message  = "a_string_set_length must have length 4"
  }
  default = "a"  
}

variable "a_string_pattern_1" {
  type        = string
  description = "A string variable that must be a valid IPv4 address"
  validation {
    condition      = can( regex( "^[0-9]{1,3}(\\.[0-9]{1,3}){3}$" , var.a_string_pattern_1 ) ) 
    error_message  = "a_string_pattern_1 must be an IPv4 address"
  }
  default = "1.1.1.1"
}

variable "a_string_pattern_2" {
  type        = string
  description = "string that must be a valid colour hex code in the form #RRGGBB"
  validation {
    condition      =can(regex("^#[0-9a-fA-F]{6}$",var.a_string_pattern_2))
    error_message  = "a_string_pattern_2 must be a valid colour hex code in the form #RRGGBB"
  }
  default = "#000000"
}

variable "a_string_enum_escaped_characters_kind_1" {
  type        = string
  description = "A string variable that must some complicated escaped characters"
  validation {
    condition      = contains(["\\", "\"", "\\\"", "$${abc}","\n","\t","10%","10%%","$a","$$a","\r","\\r", null, "<", ">", "&"], var.a_string_enum_escaped_characters_kind_1)
    error_message  = "Invalid value for a_string_enum_escaped_characters"
  }
  default = "\\"
}

variable "a_string_enum_escaped_characters_kind_2" {
  type        = string
  description = "A string variable that must some complicated escaped characters"
  validation {
    condition      = var.a_string_enum_escaped_characters_kind_2 == "\\" || var.a_string_enum_escaped_characters_kind_2 == "\"" || var.a_string_enum_escaped_characters_kind_2 == "\\\"" || var.a_string_enum_escaped_characters_kind_2 == "$${abc}" || var.a_string_enum_escaped_characters_kind_2 == "\n" || var.a_string_enum_escaped_characters_kind_2 == "\t" || var.a_string_enum_escaped_characters_kind_2 == "10%" || var.a_string_enum_escaped_characters_kind_2 == "10%%" || var.a_string_enum_escaped_characters_kind_2 == "$a" || var.a_string_enum_escaped_characters_kind_2 == "$$a" || var.a_string_enum_escaped_characters_kind_2 == "\r" || var.a_string_enum_escaped_characters_kind_2 == "\\r" || var.a_string_enum_escaped_characters_kind_2 == null || var.a_string_enum_escaped_characters_kind_2 == "<" || var.a_string_enum_escaped_characters_kind_2 == ">" || var.a_string_enum_escaped_characters_kind_2  == "&"
    error_message  = "Invalid value for a_string_enum_escaped_characters"
  }
  default = "\""
}

variable "a_string_multiple_validation_conditions" {
  type = string
  description = "A string which has a minimum and maximum length, defined as 2 separate validation blocks"
  validation {
    condition = length(var.a_string_multiple_validation_conditions) < 8
    error_message = "Must be fewer than 8 characters"
  }
  validation {
    condition = length(var.a_string_multiple_validation_conditions) >= 1
    error_message = "Must have greater than or equal to 1 character (note: redundant check for test)"
  }
  validation {
    condition = length(var.a_string_multiple_validation_conditions) >= 2
    error_message = "Must have greater than or equal to 2 characters"
  }
  default = "hello"
}

variable "a_complex_condition_with_complex_error_message" {
  type        = list(string)
  description = "A list of names that must be 3-24 lowercase letters and numbers."
  validation {
    condition = alltrue([
      for name in var.a_complex_condition_with_complex_error_message :
      can(regex("^[a-z0-9]{3,24}$", name)) # 3-24 lowercase letters and numbers
    ])
    error_message = format(<<-EOT
        `var.a_complex_condition_with_complex_error_message[*]` value is invalid: %s

        A name must consist of 3-24 lowercase letters and numbers.
      EOT
      , try(join(", ", [
        for idx, name in var.a_complex_condition_with_complex_error_message : format("'%s' [%d]", name, idx)
        if !can(regex("^[a-z0-9]{3,24}$", name))
      ]), "<failed to compute>")
    )
  }
  default = []
}
