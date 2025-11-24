# DSMA5
Mandatory Activity 5 in Distributed Systems, BSc (Autumn 2025) at IT University of Copenhagen. 

## How to run
1. run to instances of replica.go. After starting, please give both a unique id. exactly one need the id '1' and exactly one need the id '2'
2. run any number of instances of client.go. After starting, please give them all a unique id.
3. use the program by typing in the commands:
- use 'bid <amount>' to make a new bid. If no action is ongoing, a new one will start and will be open for 20 seconds
- use 'result' to query what the current result of the newest auction.