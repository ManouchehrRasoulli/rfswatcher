package internal

/*
	watcher --> used for getting notification on file changes.
	file_handler --> contains a metadata about files, and path which we attempt to watch over.

	** Usage
	1 - create a watcher over a given path.
	2 - subscribe handler over given path.

	TODO : use handler in network connection session inorder to notify clients about changes, and updating them.
*/
