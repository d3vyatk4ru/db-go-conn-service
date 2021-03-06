USE master;
GO

IF DB_ID(N'db_golang') IS NOT NULL
DROP DATABASE [db_golang];
GO

CREATE DATABASE [db_golang]
ON
    (
        NAME = db_golang_dat,
        FILENAME = '/var/lib/mssqlql/data/db_golang.mdf'
    )
LOG ON
	(
		NAME = db_golang_log,
		FILENAME = '/var/lib/mssqlql/data/db_golang_log'
	)
GO

USE [db_golang];
GO

IF OBJECT_ID(N'items') IS NOT NULL
DROP TABLE [items];
GO

CREATE TABLE [items] (
  [id]          INT NOT NULL IDENTITY(1, 1) PRIMARY KEY,
  [title]       NVARCHAR(255) NOT NULL,
  [description] TEXT NOT NULL,
  [updated]     NVARCHAR(255) DEFAULT NULL
);
GO

SET IDENTITY_INSERT [items] ON;
GO

INSERT INTO
    [items] ([id], [title], [description], [updated])
VALUES
    (1,	'database/sql',	N'Conversation about data bases', 'rvasily'),
    (2,	'memcache',	N'Conversation about memorycache with practice example', 'rvasily'),
    (3,	'ml/python', N'Linear regression and noise distributed', 'ddaniil'),
    (4,	'pytorch',	N'The main topic of library', 'ddaniil'),
    (5,	'data transfer', N'UDP and TCP protocol', NULL);
GO

IF OBJECT_ID(N'users') IS NOT NULL
DROP TABLE [users];
GO

CREATE TABLE [users] (
  [user_id]  INT NOT NULL IDENTITY(1, 1) PRIMARY KEY,
  [login]    NVARCHAR(255) NOT NULL,
  [password] NVARCHAR(255) NOT NULL,
  [email]    NVARCHAR(255) NOT NULL,
  [info]     TEXT NOT NULL,
  [updated]  NVARCHAR(255) DEFAULT NULL
);
GO

INSERT INTO
    [users]([login], [password], [email], [info], [updated])
VALUES
    ('rvasily', 'love', 'rvasily@example.com', 'none', NULL),
    ('ddaniil', '9kin', 'ddaniil@example.com', 'none', NULL);
GO
