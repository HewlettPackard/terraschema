# Copyright 2024 Hewlett Packard Enterprise Development LP

variable "an_object_with_optional" {
    type = object({
        # This is A
        a = string
        b = number # This is B
        c = bool
        d = optional(string)
        # This is E
        # @example: This is a multi-line
        # example.
        e = object({
          a = string
        })
        # This is F
        # @deprecated
        f = optional(string)
        # This is G
        # @example: first example
        # @example: second example
        g = string
        # This is H
        # @example:
        # {
        #   name    = "web-server"
        #   port    = 8080
        #   enabled = true
        # }
        h = string
    })
    description = "This is a nested object with comments"
}