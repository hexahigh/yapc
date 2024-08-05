<script>
	import logo from '$lib/img/logo.svg';
</script>

<div class="flex flex-col items-center justify-center min-h-screen">
	<img src={logo} alt="logo" class="w-64 h-64" />
	<h1 class="text-3xl font-bold">YAPC has shutdown</h1>
	<div class="text-lg w-1/2 flex flex-col gap-2 mb-8">
		<p>
			YAPC started off as a side project of mine, I was surprised when I found out that other people
			had started using it. It was fun working on it at first, however, I quickly got burned out and
			stopped working on it. I quickly realized that running a file hoster is not as easy as it
			might seem.
		</p>
		<p>
			One of the main reasons I'm shutting it down is because I don't have the time to work on it
			anymore. Balancing the project with other projects i have became very difficult, and it was
			clear that YAPC required more attention than I could afford to give it.
		</p>
		<p>
			Additionally, the site faced several issues. The server was hit by DDoS attacks multiple
			times, which was a pain in the ass because it not only took the down the primary server, the
			primary server was hosted on my home network. So my entire home network was affected. People
			started crashing the server using various methods which required fixing the code, which i did
			not have time to do.
		</p>
		<p>Storage became another major issue. And i quickly ran out</p>
		<p>
			The final nail in the coffin was when i was informed that CSAM was being uploaded to YAPC,
			preventing this from happening would be hard and time consuming.
		</p>
		<p>
			In light of these challenges, and after <s>much</s> consideration, I've made the difficult decision
			to shut down YAPC. Files will remain accessible for the time being. I would love to be able to upload the files to the Internet Archive but due to the files potentially containing CSAM or other personal information, it's not feasible without thorough inspection and verification, which is way beyond my capacity to handle alone.
		</p>
	</div>
	<div class="text-lg w-1/2 flex flex-col">
		<p>
			If you are looking for another site to host your files, here is a list of some of my
			favorites:
		</p>
		<ul>
			<li>
				<p><a href="https://pixeldrain.com" target="_blank" rel="noreferrer">PixelDrain</a> Has great upload and download speeds, cheap paid plans, but it requires an account to upload to.</p>
				<p><a href="https://pomf.lain.la" target="_blank" rel="noreferrer">Pomf.lain.la</a> 7666's pomf clone. Files never expire and can have sizes of up to 1GB.</p>
				<p><a href="https://catbox.moe" target="_blank" rel="noreferrer">Catbox</a> is known for its simplicity and reliability. It offers straightforward file hosting without the need for an account, making it an excellent choice for quick uploads. Catbox supports files up to 200MB in size and keeps them forever.</p>
			</li>
		</ul>
	</div>
</div>

<style lang="postcss">
a {
	@apply underline text-blue-500;
}
</style>