import { endpoint } from '$lib/conf.js';

export async function GET({ url }) {
   // Extract the HASH and the file extension from the url
   const hash = url.searchParams.get('h') || '0';
   const ext = url.searchParams.get('e') || 'bin';
   const filename = url.searchParams.get('f') || 'file.bin';

   // Construct the URL to the file
   const fileUrl = `${endpoint}/get2/?h=${hash}&e=${ext}&f=${filename}`;

   // Create a 301 redirect response
   return new Response(null, {
       status: 301,
       headers: {
           'Location': fileUrl
       }
   });
}