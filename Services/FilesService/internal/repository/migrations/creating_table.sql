use CloudStorage_FileStorageService;
drop table files;
CREATE TABLE files (
    id int PRIMARY KEY AUTO_INCREMENT,
    hash VARCHAR(255),
    user_id int NOT NULL,
    filename VARCHAR(255) NOT NULL,
    filepath VARCHAR(255) NOT NULL UNIQUE,
    filetype VARCHAR(255) not null,
    sharestatus BOOLEAN not NULL DEFAULT False,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
