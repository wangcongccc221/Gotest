import fs from 'node:fs';
import path from 'node:path';

const [inputPath, outputPath] = process.argv.slice(2);

if (!inputPath || !outputPath) {
  console.error('Usage: node tools/convert_mysql_dump_to_sqlite.mjs <mysql-dump.sql> <sqlite-output.sql>');
  process.exit(1);
}

const input = fs.readFileSync(inputPath, 'utf8');
const tables = extractCreateTables(input);
const inserts = splitSqlStatements(input).filter((statement) => /^INSERT\s+INTO\s+/i.test(statement.trim()));

const output = [
  '-- Converted from MySQL dump for SQLite initialization.',
  '-- Source: 0505.sql',
  '-- MySQL procedures, events, CREATE DATABASE, USE, DELIMITER, charset, and storage-engine clauses are intentionally omitted.',
  'PRAGMA foreign_keys = OFF;',
  '',
  ...tables.flatMap((table) => [convertCreateTable(table), '']),
  ...inserts.flatMap((statement) => [convertInsert(statement) + ';', '']),
  'PRAGMA foreign_keys = ON;',
  '',
].join('\n');

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, output, 'utf8');

console.log(`Converted ${tables.length} tables and ${inserts.length} insert statements.`);

function extractCreateTables(sql) {
  const tables = [];
  const tablePattern = /CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+`([^`]+)`\s*\(([\s\S]*?)\n\)\s*ENGINE\s*=\s*[^;]*;/gi;
  let match;
  while ((match = tablePattern.exec(sql)) !== null) {
    tables.push({
      name: match[1],
      body: match[2],
    });
  }
  return tables;
}

function convertCreateTable(table) {
  const definitions = [];
  const indexes = [];
  let hasInlineAutoIncrementPrimaryKey = false;
  let primaryKeyDefinition = '';

  for (const rawPart of splitTopLevel(rawPartBody(table.body), ',')) {
    const part = rawPart.trim();
    if (!part) {
      continue;
    }

    if (/^(UNIQUE\s+)?KEY\s+/i.test(part)) {
      const indexSql = convertIndex(table.name, part);
      if (indexSql) {
        indexes.push(indexSql);
      }
      continue;
    }

    if (/^PRIMARY\s+KEY\s+/i.test(part)) {
      primaryKeyDefinition = convertPrimaryKey(part);
      continue;
    }

    if (/^CONSTRAINT\s+/i.test(part) || /^FOREIGN\s+KEY\s+/i.test(part)) {
      continue;
    }

    const column = convertColumn(part);
    if (column) {
      definitions.push(column.sql);
      hasInlineAutoIncrementPrimaryKey ||= column.inlinePrimaryKey;
    }
  }

  if (primaryKeyDefinition && !hasInlineAutoIncrementPrimaryKey) {
    definitions.push(primaryKeyDefinition);
  }

  const tableSql = [
    `CREATE TABLE IF NOT EXISTS ${quoteIdentifier(table.name)} (`,
    definitions.map((definition) => `  ${definition}`).join(',\n'),
    ');',
  ].join('\n');

  if (indexes.length === 0) {
    return tableSql;
  }
  return [tableSql, ...indexes].join('\n');
}

function rawPartBody(body) {
  return body.replace(/\r\n/g, '\n');
}

function convertColumn(part) {
  const match = part.match(/^`([^`]+)`\s+([\s\S]+)$/);
  if (!match) {
    return null;
  }

  const name = match[1];
  let rest = cleanColumnDefinition(match[2]);
  const autoIncrement = /\bAUTO_INCREMENT\b/i.test(rest);
  rest = rest.replace(/\bAUTO_INCREMENT\b/gi, '').trim();

  const typeMatch = rest.match(/^([a-zA-Z]+)(?:\s*\([^)]*\))?/);
  if (!typeMatch) {
    return {
      sql: `${quoteIdentifier(name)} TEXT`,
      inlinePrimaryKey: false,
    };
  }

  const mysqlType = typeMatch[1].toLowerCase();
  let constraints = rest.slice(typeMatch[0].length).trim();
  constraints = cleanConstraints(constraints);

  if (autoIncrement) {
    return {
      sql: `${quoteIdentifier(name)} INTEGER PRIMARY KEY AUTOINCREMENT${appendNonDefaultConstraints(constraints)}`,
      inlinePrimaryKey: true,
    };
  }

  return {
    sql: `${quoteIdentifier(name)} ${sqliteType(mysqlType)}${constraints ? ` ${constraints}` : ''}`,
    inlinePrimaryKey: false,
  };
}

function cleanColumnDefinition(definition) {
  return definition
    .replace(/\s+CHARACTER\s+SET\s+\w+/gi, '')
    .replace(/\s+COLLATE\s+[\w_]+/gi, '')
    .replace(/\s+COMMENT\s+'(?:''|[^'])*'/gi, '')
    .replace(/\s+COMMENT\s+"(?:""|[^"])*"/gi, '')
    .replace(/\s+USING\s+BTREE/gi, '')
    .replace(/\bUNSIGNED\b/gi, '')
    .replace(/\bZEROFILL\b/gi, '')
    .replace(/\s+ON\s+UPDATE\s+CURRENT_TIMESTAMP(?:\(\))?/gi, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function cleanConstraints(constraints) {
  return constraints
    .replace(/\s+USING\s+BTREE/gi, '')
    .replace(/\bDEFAULT\s+NULL\b/gi, 'DEFAULT NULL')
    .replace(/\bCURRENT_TIMESTAMP\(\)/gi, 'CURRENT_TIMESTAMP')
    .replace(/\s+/g, ' ')
    .trim();
}

function appendNonDefaultConstraints(constraints) {
  const cleaned = constraints
    .replace(/\bNOT\s+NULL\b/gi, '')
    .replace(/\bNULL\b/gi, '')
    .replace(/\bDEFAULT\s+NULL\b/gi, '')
    .trim();
  return cleaned ? ` ${cleaned}` : '';
}

function sqliteType(mysqlType) {
  switch (mysqlType) {
    case 'int':
    case 'integer':
    case 'tinyint':
    case 'smallint':
    case 'mediumint':
    case 'bigint':
    case 'bit':
    case 'bool':
    case 'boolean':
    case 'year':
      return 'INTEGER';
    case 'float':
    case 'double':
    case 'real':
      return 'REAL';
    case 'decimal':
    case 'numeric':
      return 'NUMERIC';
    case 'blob':
    case 'longblob':
    case 'mediumblob':
    case 'tinyblob':
    case 'binary':
    case 'varbinary':
      return 'BLOB';
    default:
      return 'TEXT';
  }
}

function convertPrimaryKey(part) {
  const body = part.replace(/\s+USING\s+BTREE/gi, '').match(/\(([\s\S]+)\)/);
  if (!body) {
    return '';
  }
  return `PRIMARY KEY (${convertColumnList(body[1]).join(', ')})`;
}

function convertIndex(tableName, part) {
  const match = part.match(/^(UNIQUE\s+)?KEY\s+`([^`]+)`\s*\(([\s\S]+)\)/i);
  if (!match) {
    return '';
  }
  const unique = Boolean(match[1]);
  const indexName = `idx_${tableName}_${match[2]}`;
  const columns = convertColumnList(match[3]);
  if (columns.length === 0) {
    return '';
  }
  return `CREATE ${unique ? 'UNIQUE ' : ''}INDEX IF NOT EXISTS ${quoteIdentifier(indexName)} ON ${quoteIdentifier(tableName)} (${columns.join(', ')});`;
}

function convertColumnList(columnList) {
  return splitTopLevel(columnList, ',')
    .map((column) => column.trim().replace(/\s+USING\s+BTREE/gi, ''))
    .map((column) => column.replace(/\((\d+)\)/g, ''))
    .map((column) => column.replace(/\s+(ASC|DESC)\b/gi, ''))
    .map((column) => {
      const match = column.match(/`([^`]+)`/);
      return match ? quoteIdentifier(match[1]) : quoteIdentifier(column.trim());
    })
    .filter((column) => column !== '""');
}

function convertInsert(statement) {
  return statement
    .trim()
    .replace(/^INSERT\s+INTO\s+/i, 'INSERT OR IGNORE INTO ')
    .replace(/`([^`]+)`/g, (_, identifier) => quoteIdentifier(identifier));
}

function quoteIdentifier(identifier) {
  return `"${String(identifier).replaceAll('"', '""')}"`;
}

function splitTopLevel(text, delimiter) {
  const parts = [];
  let current = '';
  let depth = 0;
  let quote = '';

  for (let index = 0; index < text.length; index++) {
    const ch = text[index];

    if (quote) {
      current += ch;
      if (ch === '\\' && quote === "'" && index + 1 < text.length) {
        current += text[++index];
        continue;
      }
      if (ch === quote) {
        if (index + 1 < text.length && text[index + 1] === quote) {
          current += text[++index];
          continue;
        }
        quote = '';
      }
      continue;
    }

    if (ch === "'" || ch === '"' || ch === '`') {
      quote = ch;
      current += ch;
      continue;
    }

    if (ch === '(') {
      depth++;
      current += ch;
      continue;
    }
    if (ch === ')') {
      depth = Math.max(0, depth - 1);
      current += ch;
      continue;
    }
    if (ch === delimiter && depth === 0) {
      parts.push(current);
      current = '';
      continue;
    }
    current += ch;
  }

  if (current.trim()) {
    parts.push(current);
  }
  return parts;
}

function splitSqlStatements(sqlText) {
  const statements = [];
  let current = '';
  let quote = '';

  for (let index = 0; index < sqlText.length; index++) {
    const ch = sqlText[index];

    if (quote) {
      current += ch;
      if (ch === '\\' && quote === "'" && index + 1 < sqlText.length) {
        current += sqlText[++index];
        continue;
      }
      if (ch === quote) {
        if (index + 1 < sqlText.length && sqlText[index + 1] === quote) {
          current += sqlText[++index];
          continue;
        }
        quote = '';
      }
      continue;
    }

    if (ch === "'" || ch === '"' || ch === '`') {
      quote = ch;
      current += ch;
      continue;
    }
    if (ch === '-' && sqlText[index + 1] === '-') {
      while (index < sqlText.length && sqlText[index] !== '\n') {
        index++;
      }
      current += '\n';
      continue;
    }
    if (ch === '/' && sqlText[index + 1] === '*') {
      index += 2;
      while (index + 1 < sqlText.length && !(sqlText[index] === '*' && sqlText[index + 1] === '/')) {
        index++;
      }
      index++;
      continue;
    }
    if (ch === ';') {
      const statement = current.trim();
      if (statement) {
        statements.push(statement);
      }
      current = '';
      continue;
    }
    current += ch;
  }

  const statement = current.trim();
  if (statement) {
    statements.push(statement);
  }
  return statements;
}
