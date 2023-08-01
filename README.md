# Chatrr

Chatrr is a command line chatbot built around the openai api with one special feature: a memory! 

## How to use it
You need to have set the environment variable `OPENAI_API_KEY` to be your openai api key. Then:
```
make build
./bin/chatrr
```

## How it works
When you submit input, it will be embedded and then used to query a [sqlite database with the vector similarity search extension](https://github.com/asg017/sqlite-vss). If the results have strong similarity, they will be constructed into a "memory", prepended to your input, and sent to the completions api. This allows the bot to do cool things like remember your name from one conversation to the next or remember details of a project you were discussing. 
![image](https://github.com/dleviminzi/chatrr/assets/51272568/6a804546-6414-404f-9369-4bb561a17493)

If you would like the bot to remember the response it gave to your last prompt type: `memorize 1`. This tells the bot to remember the last (Prompt, Response) interaction in the conversation. You can tell it to remember however many you like.
![image](https://github.com/dleviminzi/chatrr/assets/51272568/8fd8db0d-6ca3-4050-8269-9d0567010bbd)

## Some problems
1. I suspect that as conversations carry on for a long time, the bot might become confused. This effect would likely be amplified if you change topics a lot.
2. Once you've told the bot to memorize a conversation fragment it will be in there unless you delete the database.
3. This uses embedding + completions for every single prompt.
4. If you have a long conversation and ask the bot to remember the entire thing, it might exceed the context limit in future conversations (this will probably break the bot).

## Future improvements
- [x] Add timestamps to conversation fragments
- [x] Fix zero entry query sqlite vss explosion
- [ ] Make completion model configurable
- [ ] Use [tiktoken](https://github.com/pkoukk/tiktoken-go) to ensure that we don't exceed model context limitations
- [ ] Claude support
- [ ] Add feature for bot to ingest documents (stored seperately)
- [ ] Add cli flagging for experimental features
- [ ] Experiment with embedding entire conversation segment when memorizing (current strategy is ok, but why not just do the whole thing?)
- [ ] Experiment with eviction policies for memories
