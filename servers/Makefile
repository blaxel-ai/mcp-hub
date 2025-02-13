ARGS:= $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

add-server:
	@if [ -d "src/$(ARGS)" ]; then \
		echo "Error: src/$(ARGS) directory already exists"; \
		exit 1; \
	fi;
	@mkdir -p src/$(ARGS)
	@git clone https://github.com/modelcontextprotocol/servers.git
	@mv servers/src/$(ARGS)/*.ts src/$(ARGS)/
	@rm -rf servers

remove-server:
	@rm -rf src/$(ARGS)

%:
	@: