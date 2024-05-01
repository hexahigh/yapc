export const endpoint = 'https://pomf1.080609.xyz'; // Main endpoint url
export const instanceName = 'YAPC'; // instance name, shown in tos
export const domain = 'pomf.080609.xyz'; // Domain where the instance is hosted, used in the TOS
export const email = 'admin@boofdev.eu'; // Contact email. PLEASE change this if you are hosting your own instance
export const endpointList = { // list of endpoints displayed in the footer
    1: {
        name: "Local (8080)",
        url: 'http://localhost:8080',
    },
    2: {
        name: "Unlimited",
        url: 'https://tiny-cougar-22.telebit.io',
    },
    3: {
        name: "Zerotier",
        url: 'http://10.0.0.5:9066',
    },
}

export const loadEndpoints = [ // Used in the load balancer
    {
        name: 'NO1',
        url: 'https://pomf1.080609.xyz',
        lat: '59.2083',
        lon: '10.9484'
    }
]

// Stuff you probably don't need to (or should) change

export const dbEndpoint = 'https://db.080609.xyz';