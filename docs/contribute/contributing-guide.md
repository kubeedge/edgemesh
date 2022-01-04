# Contributing to EdgeMesh
Welcome to EdgeMesh. Here is the contributing guide for you.

## Getting Started
1. Fork the [repository](https://github.com/kubeedge/edgemesh) on Github.
2. Read the [setup](../guide/getting-started.md) for installation.
3. Read the [developer guide](developer-guide.md) for more detail.

## Filing issues
If you have any question for EdgeMesh, please feel free to file an issue via [NEW ISSUE](https://github.com/kubeedge/edgemesh/issues/new/choose).
The subjects of the issue can include `Question about the project`, `Bug report`, `Feature request and proposal`, `Performance issues` and so on.

## Contributor Workflow
EdgeMesh welcomes all developers, please feel free to ask questions and submit pull request.
The following is a general workflow of a contributor:
1. Fork the [EdgeMesh](https://github.com/kubeedge/edgemesh) project to your personal repository, and create a new branch from main.
2. Make commits of logical units, including docs, unit test, e2e test, and code.
3. Make sure commit message are in the proper format. (see below)
4. Before you pull request, you need to make sure pass the verifications and tests. (see below)
5. Push changes to the new branch and pull a new request to [EdgeMesh](https://github.com/kubeedge/edgemesh) project.
6. The PR must receive an approval from maintainers and pass the continuous integration in github action.

### Format of the commit message
We follow a rough convention for commit messages that is designed to answer two questions: what changed and why.
The subject line should feature the what and the body of the commit should describe the why.
The commit message should be structured as follows:
```
<type>: <what changed>
<BLANK LINE>
<optional: why this change was made>
<BLANK LINE>
<optional: footer>
```

Here is an example for you
```
feature: support cross lan communication for edgemesh

The nodes in the edge scenario are often distributed in different lan, and communication is necessary, and cross lan communication functions are very important.

Refs #12
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.
In general, we advocate the following commit message type：
- feature: A new feature
- fix: A bug fix
- test: Adding missing or correcting existing tests
- chore: Changes to the build process or auxiliary tools and libraries such as documentation generation
- docs: Documentation only changes, such as README, CHANGELOG, CONTRIBUTE
- style: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- refactor: A code change that neither fixes a bug nor adds a feature
- perf: A code change that improves performance
- ci: A code change that update the script or config file of github ci action


### Creating Pull Requests
EdgeMesh generally follows the standard [github pull request](https://help.github.com/articles/about-pull-requests/) process.
To submit a proposed change, please add the code and new test cases.
After that, run these local verifications before submitting pull request to predict the pass or
fail of continuous integration in github action.
* Run and pass `make verify`
* Run and pass `make lint`
* Run and pass `make e2e`

In addition to the above process, a bot will begin applying structured labels to your PR.  

The bot may also make some helpful suggestions for commands to run in your PR to facilitate review.
These `/command` options can be entered in comments to trigger auto-labeling and notifications.
Refer to its [command reference documentation](https://go.k8s.io/bot-commands).
