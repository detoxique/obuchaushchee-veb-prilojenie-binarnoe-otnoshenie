const form = {
    login: document.getElementById('login'),
    password: document.getElementById('password'),
    button: document.querySelector('.Button')
}

function handleinput(e, name) {
    const { value } = e.target
    if (value) {
        form[name].classList.add('filled')
    }
    else {
        form[name].classList.remove('filled')
    }
}

async function handlelogin() {
    //const data = { key1: form.login.getElementsByTagName('input')[0].value, key2: form.password.getElementsByTagName('input')[0].value };
    //alert('Вы вошли в систему')
    const username = form.login.getElementsByTagName('input')[0].value;
    const password = form.password.getElementsByTagName('input')[0].value;

    console.log('username: ' + username + ' password: ' + password);

    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message);
        alert('Login successful!');
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

form.login.oninput = (e) =>handleinput(e, 'login')
form.password.oninput = (e) =>handleinput(e, 'password')

form.button.onclick = handlelogin