# Crun handlers

All handlers in this directory include usage section in their code. Please just run a handler command like the following:

```
$ crun-handler-slack
Usage: crun-handler-slack

Crun handler for sending a report to slack.

Options:
  --url <URL>             Incoming Webhook URL. This option is required.
  --channel <channel>     Channel.
  --username <username>   Username.
  --text <text>           Text. Default 'Reported by crun-handler-slack'.
...
```

## List of handlers

* `crun-handler-slack`: Crun handler for sending a report to slack.

  ![slack](https://raw.githubusercontent.com/kohkimakimoto/crun/master/handlers/img/slack.png)

* `crun-handler-teams`: Crun handler for sending a report to microsoft teams.
