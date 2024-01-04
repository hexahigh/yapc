# YAPC deployment guide

This guide will help you set up your YAPC instance.

## Prerequisites
#### You will need:
  - [A Cloudflare account (Optional)](https://www.cloudflare.com/)
  - [Node.js](https://nodejs.org/en/)
  - [Pnpm](https://pnpm.io/)
  - [Go](https://golang.org/)


### Backend
Once these tools are installed we can start by deploying the backend.
1. Clone the repository
2. Open a terminal in the directory called backend
3. Build the backend server by running the command `go build`

Now you should have an executable named `backend` in the backend directory.
You can run the server by running `./backend` in your terminal.
If you want to customize where the server stores the files you can use the `-d` flag, for example `./backend -d ./data`.
If you want to customize the port where the server is listening for connections you can use the `-p` flag, for example `./backend -p 8080`.

### Frontend
The frontend is a bit tricker to install.
1. Clone the repository
2. Open a terminal in the directory called frontend
3. Install the dependencies by running `pnpm install`
4. Open the `svelte.config.js` file in your favorite editor and edit the following line `import adapter from '@sveltejs/adapter-cloudflare'`, if you are planning on deploying the frontend to Cloudflare pages then you can leave it as it is, however, if you are planning on deploying to your own server then you should switch the line to `import adapter from '@sveltejs/adapter-auto'`.
5. Edit the config file located in `src/lib/conf.js` and replace the `endpoint` with the URL of your backend server
6. Run the `pnpm run build` command in your terminal to build the frontend and start the server.

And you should be done!
If you need any extra help please feel free to [open an issue](https://github.com/hexahigh/yapc/issues/new)