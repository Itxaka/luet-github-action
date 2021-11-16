# Luet Github Action

This Github Action allows to build packages and create repositories with [Luet](https://github.com/mudler/luet)

The packages can be hosted either in gh-pages (examples below) or either in container registries.



## Building packages

```yaml
- name: Build packages
  uses: mudler/luet-github-action
  with:
    # tree: packages
    build: true
```