This is a simple Go program that finds the first `firstup.env` file starting from the current directory walking upward, stopping before the root folder (meaning: any `/firstup.env` file will not be read).

It will write the env vars as export statements. It will also store the list of keys set in a separate environment variable and unset those next time `firstupdotenv` is run.

The output may then look like this:

```bash
unset FOO
unset BAR
export FIRSTUPDOTENV_CURRENT_SET_ENV=FOO,BAR
export FOO=value1
export BAR=value2
```

The `.env` format is a file on the form  `key=value`. It ignores empty lines and lines starting with # and lines without an equals sign. If the same key is defined more than once, the last will win.

To install:

```bash
go install github.com/bep/firstupdotenv@latest
```

This tool is meant to be used in combination with some shell extension that triggers when you `cd` into a directory. If you use the [Z shell](https://en.wikipedia.org/wiki/Z_shell), putting this in your `.zshrc` will work:

```
autoload -U add-zsh-hook

firstupdotenv_after_cd() {
	source <(firstupdotenv)
}

add-zsh-hook chpwd firstupdotenv_after_cd
```

