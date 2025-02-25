# Copyright 2024 Hewlett Packard Enterprise Development LP

variable "an_object_with_optional" {
    type = object({
        a = string
        b = number
        c = bool
        d = optional(string)
    })
    default = {
        a = "a"
        b = 1
        c = true
    }
    description = "This is an object variable with an optional field"
}

variable "a_very_complicated_object" {
    type = object({
        a = optional(string)
        b = tuple([list(string), bool])
        c = map(list(string))
        d = object({
            a = list(list(string))
            b = number
        })
        e = tuple([string, number])
        f = set(list(string))
    })
    default = {
        b = [["a", "b", "c"], true]
        c = {
            a = ["a"]
            b = ["b"]
        }
        d = {
            a = [["a", "b"], ["c", "d"]]
            b = 1
        }
        e = ["a", 1]
        f = [["a"], ["b"], ["a", "b"]]
    }
    description = "This is a very complicated object"
}

variable "a_nested_object" {
    type = object({
        c = map(object({
            cc = map(object({
                ccc = string
            }))
        }))
    })
    description = "This is a nested object"
}