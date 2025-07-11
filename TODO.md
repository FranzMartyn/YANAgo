# TODO

## Key features

- Show Notes in /index instead of placeholders
    - Implement insert of creation date in utc in postgres at yana.NewNote()
    - Return yana.Note instead of minio.ObjectInfo in yana.GetNote()
    - Implement yana.getCreationDateOfObject()
    - Implement yana.minioObjectToYanaNote()
- Edit a Note
- Delete a Note
- Don't go back without the content after the note couldn't be created

## Quality of Life features

- Replace any use of normal errors with YanaError
    - Add error messages to /login, /register and /create-note

## Things I might add

- Actual good auth
- User Settings
- Pinned Notes
- Make the "Remember me" checkbox work

