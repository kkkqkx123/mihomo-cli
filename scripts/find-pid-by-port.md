Get-NetTCPConnection -LocalPort 9090 | Select-Object LocalAddress, LocalPort, State, OwningProcess
