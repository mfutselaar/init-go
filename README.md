# init-go

I wanted a tool to set up a new environment for coding projects automatically for me. 
Before I created this, I had to copy files around manually, which was prone to forgetting
files or actions. This tool was created to deal with this.

## Setup

1. Clone this project, `git clone git@github.com:mfutselaar/init-go`
2. Build the project, `go build` or `go build -ldflags "-w -s" init-go`
3. Add the project to your PATH, eg: `export PATH=$PATH:/home/mathijs/init-go`
4. Create a config.json in one of the following locations:

* $XDG_CONFIG_HOME/init-go/config.json
* $HOME/.config/init-go/config.json
* $HOME/.init-go/config.json
* /etc/init-go.json

## Usage

`init-go <type> <parameters>` 

`type`: Is defined in your config

`parameters`: Run `init-go -h` for a list of parameters

Type and parameters are completely optional, if the type is omitted you'll be presented with
a selector.

## Example config

```json
{
    "runner": {
        "command": "bash",
        "args": [
            "-c"
        ]
    },
    "after-commands": [
        "git init .",
        "git add .",
        "git commit -m 'init'"
    ],
    "types": [
        {
            "type": "web",
            "files": [
                ["$HOME/code/.editorconfig", ".editorconfig"]
            ]
        },
        {
            "type": "php",
            "parent": "web",
            "commands": [
                "composer init"
            ],
            "files": [
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/.gitignore", ".gitignore"]
            ]
        },
        {
            "type": "laravel",
            "commands": [
                "composer create-project laravel/laravel ."
            ],
          "parent": {
                "type": "php",
                "run-commands": false
            },
            "files": [
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/grumphp.yml", "grumphp.yml"],
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/phpstan.neon", "phpstan.neon"],
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/pint.json", "pint.json"]
            ]            
        },
        {
            "type": "go",
            "commands": [
                "go mod init mfutselaar/$(basename $PWD)",
                "echo $(basename $PWD) > .gitignore"
            ]
        },
    ]
}
```

## Configuration explained

```json
{
    "runner": {
        "command": "bash",
        "args": [
            "-c"
        ]
    }
```

This section is required and is used to define how your commands are being executed. Most people will want to use either bash, zsh or sh.


```json
{
    "after-commands": [
        "git init .",
        "git add .",
        "git commit -m 'init'"
    ],

}
```

This section is a globally executed string array of commands, and will run after your project has been generated. You can omit this by
adding `--skip-after-commands` or `-s` to the command when creating a new project, eg. `init-go php --skip-after-commands`

```json
{
	"types": [
        {
            "type": "web",
            "files": [
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/.editorconfig", ".editorconfig"]
            ]
        },
        {
            "type": "php",
            "parent": "web",
            "commands": [
                "composer init"
            ],
            "files": [
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/.gitignore", ".gitignore"]
            ]
        },
        {
            "type": "laravel",
            "commands": [
				"composer create-project laravel/laravel ."
            ],
            "parent": {
                "type": "php",
                "run-commands": false
            },
            "files": [
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/grumphp.yml", "grumphp.yml"],
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/phpstan.neon", "phpstan.neon"],
                ["https://raw.githubusercontent.com/mfutselaar/codequality-laravel/refs/heads/main/pint.json", "pint.json"]
            ]            
        },
        {
            "type": "go",
            "commands": [
                "go mod init mfutselaar/$(basename $PWD)",
                "echo $(basename $PWD) > .gitignore"
            ]
        }
    ]
}
```

This section defines all the project types. Each type consists of the following properties:

* { `"type"`: `string` } : The name of the type, for example "go" or "php"
* { `"files"`: `string[][source, target]` } : This contains an array of all the files you wish copy over. If the source starts with `http` or `https`, 
the tool will assume this is a hyperlink and will try to download the files. Otherwise it will copy the file using the same mask as the source has.
* { `"commands"`: `string[]` } : A string array of commands to execute, these will be executed using the configured `runner`
* { `"parent"`: `string|object` } : This accepts both a string and an object. The string refers to another `type`. When this value has been set 
the commands and files from the refered type will be applied to this type as well. This happens before the files and commands assigned to this
type have been copied and executed. You can pass an object of { `"type"`: `string`, `"run-commands"`: `boolean` } to disable running the commands.


## Notes

This project was initially created to fiddle around with Go, and thus is not optimized and full of code-greatness. 
