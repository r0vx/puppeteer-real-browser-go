const queryString = window.location.search;
const urlParams = new URLSearchParams(queryString);

var discordtoken=urlParams.get("discordtoken")

if(discordtoken)
{
	localStorage.clear();
	localStorage.setItem('token', `"${discordtoken.replace('"', '')}"`);
	window.location.replace('https://discord.com/channels/@me');
}