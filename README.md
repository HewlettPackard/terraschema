# TerraSchema

TerraSchema (or `terraschema`) is a CLI tool which scans Terraform configuration (`.tf`)
files, parses a list of variables along with their type and validation rules, and converts
them to a schema which complies with 
[JSON Schema Draft-07](https://json-schema.org/draft-07/json-schema-release-notes).

### Installation

To install this application, do
```
$ go install github.com/HewlettPackard/terraschema@latest
```
or alternatively, download the correct binary for your PC from the [releases](https://github.com/HewlettPackard/terraschema/releases) tab.

### Motivation

JSON Schema files can be used to validate a `.tfvars.json` file without the need to run `terraform plan`. 
Some applications can also use JSON schema to generate a web form to enter variables, such as 
[react-jsonschema-form](https://github.com/rjsf-team/react-jsonschema-form). To validate a JSON file
against a schema, [santhosh-tekuri/jsonschema](https://github.com/santhosh-tekuri/jsonschema) can be
used, for example:

```
$ go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest
$ jv schema.json input.tfvars.json
```

Once a valid `.tfvars.json` file has been created, it can then be used in terraform with
```
$ terraform plan -var-file="input.tfvars.json"
$ terraform apply -var-file="input.tfvars.json"
$ terraform destroy -var-file="input.tfvars.json"
```
(see [Terraform Input Variables](https://developer.hashicorp.com/terraform/language/values/variables#variable-definitions-tfvars-files)).

In the majority of use-cases, any input file which validates against the JSON Schema 
generated by this application will also be a valid input file to the terraform module itself.
The main exceptions to this are certain classes of validation rules, which JSON Schema
do not directly support (see Custom Validation Rules).

# CLI Usage

The default behaviour of TerraSchema is to scan the current directory for Terraform 
configuration files, and create a file called `schema.json` at the same location. It 
returns an error if no Terraform configuration files are present, or if no variables 
are defined within those files. Variable are marked as `required` in the schema only 
if they don't have a default value set, and additional variables are permitted. 

Note: an `input.tfvars.json` file with additional variables (ones which don't correspond
to an existing variable in the Terraform configuration) will generate a warning when
running Terraform.

### Flags

- `-h`, `--help`: Print help instructions.
  
- `-i`, `--input`: The root directory of the terraform module. Note: only files contained
  in the root folder will be scanned. This is consistent with the behaviour of terraform modules.

- `-o`, `--output`: The output file for the schema file. Should be in the form `path/to/schema.json`.

- `--allow-empty`: Allow an empty schema (`{}`) to be created if no `.tf` files or variables are found.

- `--disallow-additional-properties`: Set additionalProperties to false in the root object and nested objects
  (see [JSON Schema definition](https://json-schema.org/understanding-json-schema/reference/object#additionalproperties)).

- `--nullable-all`: Change the default value for `nullable` in a Variable block to 'true'. This is to make the behaviour more closely reflect Terraform's own validation. See 'Nullable Variables' below.

- `--overwrite`: Allow overwriting an existing file at the output location.

- `--debug`: Print debug logs for variable retrieval and errors related to custom validation rules.

- `--stdout`: Print schema to stdout and prevent all other logging unless an error occurs. Does not create a file.
  Overrides `--debug` and `--output`.

- `--export-variables`: Export the variables in JSON format directly and do not create a JSON Schema. This provides similar functionality to applications such as terraform-docs, where the input variables can be output to a machine-readable format such as JSON. The `type` field is converted to a type constraint based on the type definition, and the `default` field is translated to its literal value. `condition` inside the `validation` block is left as a string, because it is difficult to represent arbitrary (ie unevaluated) HCL Expressions in JSON.

- `--escape-json`: Escape special characters in the JSON (`<`,`>` and `&`) so that the schema can be used in a web context. By default, this behaviour is disabled so the JSON file can be read more easily, though it does not effect external programs such as `jq`.

# Design

### Parsing Terraform Configuration Files
Parsing Terraform files is done using the [HCL package](https://github.com/hashicorp/hcl). Initially, the plan was to use an existing application such as [terraform-docs](https://github.com/terraform-docs/terraform-docs/) to preform the parsing step, but some of the fields of the `variable` block weren't implemented, such as validation rules.

TerraSchema parses each Terraform configuration file as a HCL (HashiCorp Configuration Language) file and picks out any blocks which match the definition of an input variable in Terraform. A typical `variable` block looks like this:

```hcl
variable "age" {
    type        = number
    default     = 10
    description = "Your age"
    nullable    = false
    sensitive   = false
    validation {
        condition      = var.age >= 0
        error_message  = "Age must not be negative"
    }
}
```

Note: All of these fields are optional.

This `variable` is translated into the following format in the `reader` package, so that it can be used by the rest of the application:

```Go
type VariableBlock struct {
    Type        hcl.Expression // or nil
    Default     hcl.Expression // or nil
    Description *string       
    Nullable    *bool         
    Sensitive   *bool
    Validation  *struct{
        Condition    hcl.Expression
        ErrorMessage string
    }
}
```

Empty expressions (such as `Type` and `Default`) are filtered out by the `reader` package after unmarshalling the `variable` block by setting them to nil.

This struct is then passed to the JSON Schema package so that it can create a schema based on these variable definitions. 

Here is an example schema generate from a module with only the variable listed above. More examples of generated schema files can be found in the `test` folder.

```JSON
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    // can be overridden with `--disallow-additional-properties`
    "additionalProperties": true,
    "properties": {
        "age": {
            "description": "Your age",
            "default": 10,
            "minimum": 0,
            "type": "number",
        },
    },
    "required": [] // only variables without a default are required, unless `--require-all` is set
}
```

Alternatively, if the program is run with the `--export-variables` flag, the returned JSON will be in the form:

```JSON
{
    "age": {
        "description": "Your age",
        "default": 10,
        "sensitive": false,
        "nullable": false,
        "validation": {
            "condition": "var.age >= 0",
            "error_message": "Age must not be negative"
        },
        "type": "number"
    }
}
```

### Translating Types to JSON Schema
Translation of types to Terraform is done in 2 steps. The first step is to take the `hcl.Expression` for the type from the `VariableBlock` struct, and use [go-cty](https://github.com/zclconf/go-cty/) to convert it to a 'type constraint', which is a JSON blob representing all the information about the type in a more machine-readable format.

The second phase is taking that type information and converting it to a JSON Schema definition. All types used by Terraform currently are supported here. Here is how each of them is represented. Also see [Terraform Input Variables](https://developer.hashicorp.com/terraform/language/values/variables#type-constraints) for more information on Terraform input variable types.

#### string
```json
{
    "type": "string"
}
```

#### number
```json
{
    "type": "number"
}
```

#### bool
```json
{
    "type": "boolean"
}
```

#### list(\<TYPE>)
```json
{
    "type": "array",
    "items": {
        "type": "<TYPE>"
    }
}
```

#### set(\<TYPE>)
```json
{
    "type": "array",
    "items": {
        "type": "<TYPE>"
    },
    "uniqueItems": true
}
```

#### map(\<TYPE>)
```json
{
    "type": "object",
    "additionalProperties": {
        "type": "<TYPE>"
    }
}
```

#### object({\<NAME> = \<TYPE>,... })
```json
{
    // can be overridden with `--disallow-additional-properties`
    "additionalProperties": true,
    "type": "object",
    "properties": {
        "<NAME>": {
            "type": "<TYPE>"
        },
        ...
    },
    "required": [
        "<NAME>",
        ...
    ]
}
```

#### tuple(\<TYPE 1>, ... \<TYPE N>)
```json
{
    "type": "array",
    "items": [
        {
            "type": "<TYPE 1>"
        },
        ...
        {
            "type": "<TYPE N>"
        }
    ],
    "minItems": N,
    "maxItems": N
}
```

Additionally, any nesting of these types is also valid, and will create a schema according to these rules.

---

Issue: [Optional Type Attributes](https://developer.hashicorp.com/terraform/language/expressions/type-constraints#optional-object-type-attributes) are not fully supported by go-cty (as of v1.15.0), and the program will error if it encounters a type of the form

```hcl
type = optional(<TYPE>,<DEFAULT-VALUE>)
```
with the following error:

```
Invalid type specification; Optional attribute modifier expects only one argument: the attribute type.
```

Optional declarations of the form `optional(<TYPE>)` are supported.

### Custom Validation Rules

A subset of common validation patterns have been implemented. If a validation rule is present and can't be converted to an existing rule, then the application will print a warning. The current list of valid validation rules for a variable with the name `name` is as follows:

| Condition                                                | Variable Type          | JSON Output                                        |
| -------------------------------------------------------- | ---------------------- | -------------------------------------------------- |
| **Enum conditions**                                      |                        |                                                    |
| `var.name == 1 \|\| 2 == var.name \|\| ...`              | any                    | `{"enum": [1, 2, ...]}`                            |
| `contains([1,2,...], var.name)`                          | any                    | `{"enum": [1, 2, ...]}`                            |
| **Regex conditions**                                     |                        |                                                    |
| `can(regex("<pattern>", var.name))`                      | `string`               | `{"pattern": "<pattern>"}`                         |
| **Number value comparison conditions**                   |                        |                                                    |
| `var.name < 10 && var.name > 0 && ...`                   | `number`               | `{"exclusiveMinimum": 0", "exclusiveMaximum": 10}` |
| `var.name <= 10 && var.name >= 0 && ...`                 | `number`               | `{"minimum": 0, "maximum": 10"}`                   |
| **String length comparison conditions**                  |                        |                                                    |
| `length(var.name) < 10 && length(var.name) > 0 && ...`   | `string`               | `{"minLength": 1,"maxLength": 9}`                  |
| `length(var.name) <= 10 && length(var.name) >= 0 && ...` | `string`               | `{"minLength": 0, "maxLength": 10, }`              |
| `length(var.name) == 5 && ...`                           | `string`               | `{"minLength": 5, "maxLength": 5"}`                |
| **Object length comparison conditions**                  |                        |                                                    |
| `length(var.name) < 10 && length(var.name) > 0 && ...`   | `map`, `object`        | `{"minProperties": 1,"maxProperties": 9}`          |
| `length(var.name) <= 10 && length(var.name) >= 0 && ...` | `map`, `object`        | `{"minProperties": 0, "maxProperties": 10, }`      |
| `length(var.name) == 5 && ...`                           | `map`, `object`        | `{"minProperties": 5, "maxProperties": 5"}`        |
| **Array length comparison conditions**                   |                        |                                                    |
| `length(var.name) < 10 && length(var.name) > 0 && ...`   | `list`, `tuple`, `set` | `{"minItems": 1,"maxItems": 9}`                    |
| `length(var.name) <= 10 && length(var.name) >= 0 && ...` | `list`, `tuple`, `set` | `{"minItems": 0, "maxItems": 10, }`                |
| `length(var.name) == 5 && ...`                           | `list`, `tuple`, `set` | `{"minItems": 5, "maxItems": 5"}`                  |

### Nullable Variables

If `nullable` is true in the `variable` block, then the JSON Schema will be modified to look like this. This method is primarily chosen for compatibility with react-jsonschema-form.

```JSON
"<NAME>": {
    "anyOf": [
        {
            "title": "null",
            "type": "null"
        },
        {
            "title": "<TYPE>",
            "type": "<TYPE>"
        }
    ],
    "description": "<DESCRIPTION>",
    "default": "<DEFAULT>",
    "title": "<NAME>: Select a type"
},
```

This is actually a slight behaviour change from the validator used by terraform. If `nullable` is unset, then terraform treats them as `nullable` by default. I chose not to implement that default behaviour here and instead am making the Terraform module author specify `nullable = true`. This is because otherwise schema definitions for simple programs would have to become a lot more verbose just to handle this case.

For behaviour more consistent with Terraform, the flag `--nullable-all` can be used to reset the default value for nullable to be true. Note: this rule only applies to variables which have not explicitly set the value of nullable themselves. See [Terraform documentation on nullable](https://developer.hashicorp.com/terraform/language/values/variables#disallowing-null-input-values
).

As an example, here is a Terraform configuration file which does not specify `nullable`:

```hcl
variable "name" {
    type = string
    nullable = false
}

variable "age" {
    type = number
    default = 10
}
```

Without `--nullable-all`, this would result in the following JSON Schema file: 

```json
{
    "additionalProperties": true,
    "properties": {
        "age": {
            "default": 10,
            "type": "number"
        },
        "name": {
            "type": "string"
        }
    },
    "required": [
        "age",
        "name"
    ]
}
```

And if `--nullable-all` is set to true, then the 'default' value for nullable will be true, so the schema will change to reflect this:

```json
{
    "additionalProperties": true,
    "properties": {
        "age": {
            "anyOf": [
                {
                    "title": "null",
                    "type": "null"
                },
                {
                    "title": "number",
                    "type": "number"
                }
            ],
            "default": 10,
            "title": "age: Select a type"
        },
        "name": {
            "type": "string"
        }
    },
    "required": [
        "age",
        "name"
    ]
}
```

`name` is not affected here since it has `nullable = false` in its HCL definition.

### Default Handling

Default handling is relatively straightforward. The default specified in Terraform is rendered to a JSON object, and added to the default field in the JSON Schema. Type checking is not performed on the default value. This is in line with how the JSON Schema creators generally expect this field to be used. See their notes on [annotations](https://json-schema.org/understanding-json-schema/reference/annotations#:~:text=The%20default%20keyword%20specifies%20a%20default%20value.).