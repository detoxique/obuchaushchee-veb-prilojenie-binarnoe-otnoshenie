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

form.login.oninput = (e) =>handleinput(e, 'login')
form.password.oninput = (e) =>handleinput(e, 'password')
form.button.onclick = () => alert('Вы вошли в систему')