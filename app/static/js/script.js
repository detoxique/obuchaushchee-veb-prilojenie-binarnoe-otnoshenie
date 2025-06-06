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
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage

    // if (form.login.value != "") {
    //     form.login.classList.add('filled');
    // }

    // if (form.password.value != "") {
    //     form.password.classList.add('filled');
    // }
    
    if (!token) {
        // Токена нет
        console.log("No token")
        return;
    }

    form[login].classList.add('filled')
    form[password].classList.remove('filled')

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:9293/api/verify', {
        method: 'POST',
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
        //document.body.innerHTML = `<h2>Logged in</h2>`;
        window.location.href = '/profile';
    })
    .catch(error => {
        // Ошибка проверки токена
        //console.error('Error:', error);
        localStorage.removeItem('access_token'); // Удаляем недействительный токен
        //localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
        //setTimeout(() => window.location.href = '/', 5000);

        fetch('http://localhost:9293/api/refreshtoken', {
            method: 'POST',
            headers: {
            'Authorization': localStorage.getItem('refresh_token'), // Передаем токен в заголовке
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
            // Сохраняем токен в localStorage
            localStorage.setItem('access_token', data.access_token);
            localStorage.setItem('refresh_token', data.refresh_token);
            window.location.href = '/profile';
        })
        .catch(error => {
            // Ошибка проверки токена
            console.error('Error:', error);
            localStorage.removeItem('access_token'); // Удаляем недействительный токен
            localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
        });
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
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        
        window.location.href = '/profile';
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

form.login.oninput = (e) =>handleinput(e, 'login')
form.password.oninput = (e) =>handleinput(e, 'password')

form.button.onclick = handlelogin