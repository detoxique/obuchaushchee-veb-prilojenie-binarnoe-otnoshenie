const form = {
    login: document.getElementsByName('username'),
    password: document.getElementsByName('password'),
    role: document.querySelector("#role"),
    group: document.querySelector("#groupname")
}

document.addEventListener('DOMContentLoaded', function() {
    
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }

    // Отправляем токен на сервер для проверки
    fetch('http://localhost:9293/api/getadminpaneldata', {
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
        return response.text();
    })
    .then(data => {
        // Страница получена успешно
        console.log(data)
        document.body.innerHTML = data
    })
    .catch(error => {
        // Ошибка проверки токена
        document.body.innerHTML = "<h2>Нет доступа<h2>"
        setTimeout(function() {
            window.location.href = '/';
        }, 2000)
    });
});



const groupForm = {
    group: document.getElementsByName('groupname-input')
}

function handleAddUser() {
    const Username = document.getElementsByName('username')[0].value;
    const Password = document.getElementsByName('password')[0].value;
    const Role = document.getElementsByName('role')[0].value;
    const Groupname = document.getElementsByName('groupname')[0].value;

    fetch('http://localhost:9293/api/admin/adduser', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ Username, Password, Role, Groupname})
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Token invalid or expired');
        }
        return response.json();
    })
    .then(data => {
        // Страница получена успешно
        
        location.reload();
    })
    .catch(error => {
        // Ошибка проверки токена
        location.reload();
    });
}

function handleAddGroup() {
    const GroupName = document.getElementsByName('groupname-input')[0].value;
    console.log(GroupName);

    fetch('http://localhost:9293/api/admin/addgroup', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ GroupName})
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Token invalid or expired');
        }
        return response.json();
    })
    .then(data => {
        // Страница получена успешно
        location.reload();
    })
    .catch(error => {
        // Ошибка проверки токена
        location.reload();
    });
}

function logout() {
    localStorage.removeItem('access_token'); // Удаляем токен
    localStorage.removeItem('refresh_token'); // Удаляем токен
    window.location.href = 'http://localhost:9293/';
}

// Нажатия кнопок
document.addEventListener('click', function(event) {
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    if (event.target.tagName === 'BUTTON' && event.target.id) {
        const buttonId = event.target.id;
        
        // Проверяем, начинается ли id с "delete-idgroup-"
        if (buttonId.startsWith('delete-idgroup-')) {
            const Id = buttonId.replace('delete-idgroup-', '');
            console.log('Нажата кнопка удаления группы, ID:', Id);
            fetch('http://localhost:8080/api/admin/deletegroup', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ Id})
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Token invalid or expired');
                }
                return response.json();
            })
            .then(data => {
                // Группа удалена успешно
                location.reload();
            })
            .catch(error => {
                // Ошибка проверки токена
                location.reload();
            });
        } 
        // Проверяем, начинается ли id с "delete-user-"
        else if (buttonId.startsWith('delete-user-')) {
            const Username = buttonId.replace('delete-user-', '');
            console.log('Нажата кнопка удаления пользователя, имя:', Username);
            fetch('http://localhost:9293/api/admin/deleteuser', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ token, Username})
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Error');
                }
                return response.json();
            })
            .then(data => {
                // Группа удалена успешно
                location.reload();
            })
            .catch(error => {
                // Ошибка проверки токена
                location.reload();
            });
        }
    }
});