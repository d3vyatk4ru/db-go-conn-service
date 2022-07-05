USE master;
GO

IF DB_ID(N'items') IS NOT NULL
DROP TABLE [items];
GO

CREATE TABLE [items] (
  [id]          INT NOT NULL IDENTITY(1, 1),
  [title]       NVARCHAR(255) NOT NULL,
  [description] TEXT NOT NULL,
  [updated]     NVARCHAR(255) DEFAULT NULL,
  PRIMARY KEY ([id])
);
GO

SET IDENTITY_INSERT dbo.[items] ON;
GO

INSERT INTO
    [items] ([id], [title], [description], [updated])
VALUES
    (1,	'database/sql',	'Рассказать про базы данных',	'rvasily'),
    (2,	'memcache',	'Рассказать про мемкеш с примером использования', NULL);
GO

IF DB_ID(N'users') IS NOT NULL
DROP TABLE [users];
GO

CREATE TABLE [users] (
  [user_id]  INT NOT NULL IDENTITY(1, 1) PRIMARY KEY,
  [login]    NVARCHAR(255) NOT NULL,
  [password] VARCHAR(255) NOT NULL,
  [email]    VARCHAR(255) NOT NULL,
  [info]     TEXT NOT NULL,
  [updated]  VARCHAR(255) DEFAULT NULL
);
GO

INSERT INTO
    [users]([login], [password], [email], [info], [updated])
VALUES
    ('rvasily', 'love', 'rvasily@example.com', 'none', NULL);
GO
