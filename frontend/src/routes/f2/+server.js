import { endpoint, loadEndpoints as servers } from '$lib/conf.js';

export async function GET({ url, request }) {
	// Extract the HASH and the file extension from the url
	const hash = url.searchParams.get('h') || '0';
	const ext = url.searchParams.get('e') || 'bin';
	const filename = url.searchParams.get('f') || 'file.bin';

	let fileUrl;

	// Select the least loaded server
	const leastLoadedServer = await selectLeastLoadedServer(servers);

	// Construct the URL to the file on the default server
	fileUrl = `${leastLoadedServer}/get2/?h=${hash}&e=${ext}&f=${filename}`;

	// Create a 301 redirect response
	return new Response(null, {
		status: 301,
		headers: {
			Location: fileUrl
		}
	});
}

async function selectLeastLoadedServer(servers, timeout = 2000) {
	const loads = await Promise.all(
		servers.map(async (server) => {
			try {
				const response = await fetchWithTimeout(`${server.url}/load`, {}, timeout);
				if (!response.ok) {
					throw new Error('Failed to fetch load');
				}
				const data = await response.json();
				return { server: server.name, load: data.uploads + data.downloads };
			} catch (error) {
				console.error(`Failed to fetch load from ${server.name}:`, error);
				return { server: server.name, load: Infinity }; // Return a high value to penalize this server
			}
		})
	);

	// Check if all servers have failed (i.e., all loads are Infinity)
	const allFailed = loads.every((load) => load.load === Infinity);

	if (allFailed) {
		// Fallback to the default server
		console.log('All servers failed to respond. Falling back to the default server.');
		return endpoint;
	}

	let leastLoadedServer = servers[0].name;
	let minLoad = loads[0].load;

	for (let i = 1; i < loads.length; i++) {
		if (loads[i].load < minLoad) {
			leastLoadedServer = loads[i].server;
			minLoad = loads[i].load;
		}
	}

	// Find the server object for the least loaded server
	const selectedServer = servers.find((server) => server.name === leastLoadedServer);

	return selectedServer.url; // Return the URL of the least loaded server
}

// Function to calculate the distance between two points using the Haversine formula
function calculateDistance(lat1, lon1, lat2, lon2) {
	const R = 6371; // Radius of the earth in km
	const dLat = deg2rad(lat2 - lat1);
	const dLon = deg2rad(lon2 - lon1);
	const a =
		Math.sin(dLat / 2) * Math.sin(dLat / 2) +
		Math.cos(deg2rad(lat1)) * Math.cos(deg2rad(lat2)) * Math.sin(dLon / 2) * Math.sin(dLon / 2);
	const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
	const d = R * c; // Distance in km
	return d;
}

function deg2rad(deg) {
	return deg * (Math.PI / 180);
}

// Function to send a request with a timeout
async function fetchWithTimeout(url, options, timeout = 2000) {
	return Promise.race([
		fetch(url, options),
		new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), timeout))
	]);
}
