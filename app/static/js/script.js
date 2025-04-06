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

document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('jwtToken'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token")
        return;
    }

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:8080/api/verify', {
        method: 'GET',
        headers: {
            'Authorization': token, // Передаем токен в заголовке
            'Content-Type': 'application/json'
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Token invalid or expired');
        }
        return response.json();
    })
    .then(data => {
        // Успешная проверка токена
        console.log(data.message); // Например: "Token valid for user: admin"
        document.body.innerHTML = `<h2>${data.message}</h2>`;
    })
    .catch(error => {
        // Ошибка проверки токена
        console.error('Error:', error);
        document.body.innerHTML = '<h2>Ошибка авторизации. Токен недействителен.</h2>';
        localStorage.removeItem('jwtToken'); // Удаляем недействительный токен
        setTimeout(() => window.location.href = '/index.html', 2000);
    });
});


async function handlelogin() {
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

        // Сохраняем токен в localStorage
        localStorage.setItem('jwtToken', data.token);
        alert('Login successful!');
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

form.login.oninput = (e) =>handleinput(e, 'login')
form.password.oninput = (e) =>handleinput(e, 'password')

form.button.onclick = handlelogin