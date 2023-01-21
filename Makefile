all:
	$(MAKE) wows-recruiting-bot wows-recruiting-bot-static

wows-recruiting-bot: $(SOURCES)
	go build

wows-recruiting-bot-static: $(SOURCES)
	CGO_ENABLED=0 go build -ldflags "-s -w" -o wows-recruiting-bot-static

test:
	go test

clean:
	rm -f wows-recruiting-bot
	rm -f wows-recruiting-bot-static

.PHONY: clean test clean-all all
