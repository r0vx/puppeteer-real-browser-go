if (document.querySelector('.discord-token-login-popup')) {
    document.querySelector('#submit').addEventListener('click', () => {
        token = document.querySelector('#token').value;

        if (token != '') {
            document.querySelector('#token').style.border = '1px solid #5865f2';
            window.open("https://discord.com?discordtoken="+token, '_blank');
        } else {
            document.querySelector('#token').style.border = '1px solid #5865f2';
        }
    })
}
