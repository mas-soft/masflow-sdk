.PHONY: generate clean

generate:
	buf generate

clean:
	rm -rf internal/activitypb/activity/
