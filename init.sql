-- Criar tabela pessoa se não existir
-- Estrutura real: id_pessoa como chave primária, cod_identificador como campo separado
CREATE TABLE IF NOT EXISTS pessoa (
    id_pessoa SERIAL PRIMARY KEY,
    cod_identificador INTEGER NOT NULL,
    cpf VARCHAR(14),
    funcao VARCHAR(100),
    status BOOLEAN DEFAULT true
);

-- Criar índice para melhorar performance nas consultas
CREATE INDEX IF NOT EXISTS idx_pessoa_cod_identificador ON pessoa(cod_identificador);



