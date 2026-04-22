# Ideia

A ideia dessa solução é levar para as interfaces web das IAs mais populares (ChatGPT, Claude, Gemini, etc) uma forma de se conectar ao dispositivo local para captura de contexto especifico do projeto do usuario. Dessa forma, muito do comportamento de agentes locais de codigicação (Ex: Claude Code, Codex) podem ser levadas para as aplicações web. 

## Recorte

No inicio, apenas as ferramentas uteis para captura de contexto, sem execução de código remota, patch em arquivos, criação de diretórios, etc. O modelo vai enxergar uma codebase readonly e propor as soluções no chat para o usuario realizar. 

## Arquitetura proposta

Eu vejo dois componentes principais. O primeiro é uma ponte entre os dois sistemas. 
- Um servidor em Python que expõe um MCP para acesso do agente: Ele recebe comandos do modelo via MCP
- Um executor local (Golang) que proativamente abre uma conexão com o servidor em python, executa os comandos e devolve a saida para o servidor

## Fluxo proposto

(ChatGPT) -- {Tree(".")} --> Servidor MCP <--- {Tree(".") -> { "response": {"app/... utils/..."} }}----> Executor Local

## Ferramentas V1:

-- Read
-- Grep
-- Dir
-- Tree