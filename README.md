# craftp
`craftp` is a `git`-like sftp client written in golang using the `sftp`-package. It also tries to replace the `sitecopy` package of linux.

`git`-like mainly refers to the staging-area of git. Before pushing your changes to the sftp server, you have to add and commit them. Also, rather then overwriting files while pulling from the server, the files will be merged. However, there aren't any version control features implemented.

## installation and usage

`craftp` can be installed over the usual go way: 

```
go get htts://github.com/craftamap/craftp
```

However, with every release, the binary will be pushed to the [releases page](https://github.com/craftamap/craftp/releases).

## roadmap

1. get the base functionality working (pushing and pulling files, command parsing)
2. staging-area
3. merging

## contributing
We welcome pull requests, bug fixes and issue reports.

Before proposing a large change, first please discuss your change by raising an issue.
