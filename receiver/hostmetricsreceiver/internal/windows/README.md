# winperfcounters readme

This package provides a thin wrapper around pdh.dll for querying Windows Performance counters.

This was forked from the Telegraf win_perf_counters plugin: https://github.com/influxdata/telegraf.

Original Authors:

* Alexander Neumann <an2048@googlemail.com>
* Joseph Watson <jtwatson@linux-consulting.us>
* Kevin Pors <krpors@gmail.com>

## Troubleshooting

If you are getting an error about an invalid counter, use the `typeperf` command to check the counter path
on the command line.
E.g. `typeperf "Process(chrome*)\% Processor Time"`

If no metrics are emitted even with the default config, you may need to repair
your performance counters.

1. Launch the Command Prompt as Administrator (right click Runs As Administrator).
1. Drop into the C:\WINDOWS\System32 directory by typing `C:` then `cd \Windows\System32`
1. Rebuild your counter values, which may take a few moments so please be
   patient, by running:
   ```
   lodctr /r
   ```
