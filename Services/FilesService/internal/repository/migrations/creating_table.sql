use CloudStorage_FileStorageService;
drop table files;
CREATE TABLE files (
    id int PRIMARY KEY AUTO_INCREMENT,
    hash VARCHAR(255),
    user_id int NOT NULL,
    filename VARCHAR(255) NOT NULL,
    filetype VARCHAR(255) not null,
    size BIGINT not null,
    folder_id int,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    sharestatus BOOLEAN not NULL DEFAULT False,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

drop table folders;
CREATE TABLE folders (
    id int PRIMARY KEY AUTO_INCREMENT,
    hash VARCHAR(255),
    user_id int not NULL,
    folder_name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(255) not null UNIQUE,
    parent_id int, 
    FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
