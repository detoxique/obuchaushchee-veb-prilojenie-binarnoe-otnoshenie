document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('access_token'); // Получаем токен из localStorage
    
    if (!token) {
        // Токена нет
        console.log("No token");
        window.location.href = '/';
        return;
    }

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
        })
        .catch(error => {
            // Ошибка проверки токена
            console.error('Error:', error);
            localStorage.removeItem('access_token'); // Удаляем недействительный токен
            localStorage.removeItem('refresh_token'); // Удаляем недействительный токен
        });
    });

    // Запрос данных с сервера
    fetch('http://localhost:9293/api/getteachercoursesdata', {
        method: 'POST',
        headers: {
            'Authorization': token, // Передаем токен в заголовке
            'Content-Type': 'application/json'
        }
    })
        .then(response => response.json())
        .then(data => {
            console.log(data.Courses)
            console.log(data.Groups)
            fillCoursesTable(data.Courses);
            initGroupDropdown(data.Groups);
        })
        .catch(error => console.error('Ошибка:', error));

    // Заполнение таблицы курсов
    function fillCoursesTable(courses) {
        const tbody = document.querySelector('table tbody');
        tbody.innerHTML = courses.map(course => `
            <tr data-course-id="${course.id}">
                <td>${course.Name}</td>
                <td>
                    <button class="btn btn-outline-primary btn-sm" 
                            data-bs-toggle="modal" 
                            data-bs-target="#theoryModal${course.id}">
                        Просмотр
                    </button>
                    
                    <!-- Модальное окно для теории -->
                    <div class="modal fade" id="theoryModal${course.id}" tabindex="-1">
                        <div class="modal-dialog modal-lg">
                            <div class="modal-content">
                                <div class="modal-header">
                                    <h5 class="modal-title">Материалы курса: ${course.Name}</h5>
                                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <h6>Загруженные файлы:</h6>
                                    <ul class="list-group">
                                        ${course.Files.map(file => `
                                            <li class="list-group-item d-flex justify-content-between align-items-center">
                                                ${file.Name}
                                                <small class="text-muted">
                                                    ${new Date(file.UploadDate).toLocaleDateString()}
                                                </small>
                                            </li>
                                        `).join('')}
                                    </ul>
                                </div>
                                <div class="modal-footer">
                                    <button type="button" class="btn btn-primary" 
                                            onclick="document.getElementById('fileInput${course.id}').click()">
                                        Загрузить
                                    </button>
                                    <input type="file" id="fileInput${course.id}" 
                                           style="display:none" 
                                           @change="uploadFile(${course.id})">
                                    
                                    <button type="button" class="btn btn-secondary"
                                            onclick="window.location.href='/files/choose/${course.id}'">
                                        Выбрать из загруженных
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </td>
                <td>
                    <button class="btn btn-outline-primary btn-sm" 
                            data-bs-toggle="modal" 
                            data-bs-target="#testsModal${course.id}">
                        Просмотр
                    </button>
                    
                    <!-- Модальное окно для тестов -->
                    <div class="modal fade" id="testsModal${course.id}" tabindex="-1">
                        <div class="modal-dialog modal-lg">
                            <div class="modal-content">
                                <div class="modal-header">
                                    <h5 class="modal-title">Тесты курса: ${course.Name}</h5>
                                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <ul class="list-group">
                                        ${course.Tests.map(test => `
                                            <li class="list-group-item">
                                                <div class="d-flex justify-content-between">
                                                    <div>
                                                        <h6>${test.title}</h6>
                                                        <small class="text-muted">
                                                            До ${new Date(test.end_date).toLocaleDateString()}
                                                        </small>
                                                    </div>
                                                    <div>
                                                        <span class="badge bg-info">
                                                            ${test.duration} мин
                                                        </span>
                                                    </div>
                                                </div>
                                            </li>
                                        `).join('')}
                                    </ul>
                                </div>
                                <div class="modal-footer">
                                    <a href="/test/create/${course.id}" 
                                       class="btn btn-primary"
                                       target="_blank">
                                        Создать
                                    </a>
                                    <a href="/test/choose/${course.id}" 
                                       class="btn btn-secondary"
                                       target="_blank">
                                        Выбрать из готовых
                                    </a>
                                </div>
                            </div>
                        </div>
                    </div>
                </td>
                <td><button class="btn btn-danger btn-sm">Удалить</button></td>
            </tr>
        `).join('');
    }

    // Инициализация поиска по таблице
    const searchInput = document.querySelector('.course-search input');
    searchInput.addEventListener('input', filterCoursesTable);

    function filterCoursesTable() {
        const filter = this.value.toLowerCase();
        const rows = document.querySelectorAll('table tbody tr');
        
        rows.forEach(row => {
            const courseName = row.cells[0].textContent.toLowerCase();
            row.style.display = courseName.includes(filter) ? '' : 'none';
        });
    }

    // Инициализация выпадающего списка с группами
    function initGroupDropdown(groups) {
        const excludedGroups = ['admins', 'teachers'];

        const wrapper = document.createElement('div');
        wrapper.className = 'dropdown-group-wrapper';
        wrapper.innerHTML = `
            <div class="dropdown-group">
                <input type="text" 
                       class="search-group-input" 
                       placeholder="Поиск групп..."
                       id="groupSearch">
                <div class="dropdown-group-content" id="groupDropdown">
                    ${groups
                    .filter(group => 
                        !excludedGroups.includes((group.name || '').toLowerCase()))
                    .map(group => `
                        <div class="dropdown-group-item" 
                             data-group-id="${group.id}">
                            ${group.name}
                        </div>
                    `).join('')}
                </div>
            </div>
            <div class="selected-groups"></div>
        `;

        document.querySelector('.addform').insertBefore(
            wrapper,
            document.querySelector('.addform label:nth-child(6)')
        );

        // Обработчики событий (аналогично предыдущей версии)
        const searchInput = document.getElementById('groupSearch');
        const dropdown = document.getElementById('groupDropdown');
        const selectedGroups = document.querySelector('.selected-groups');

        searchInput.addEventListener('input', function() {
            const filter = this.value.toLowerCase();
            Array.from(dropdown.children).forEach(item => {
                const text = item.textContent.toLowerCase();
                item.style.display = text.includes(filter) ? 'block' : 'none';
            });
        });

        dropdown.addEventListener('click', function(e) {
            if(e.target.classList.contains('dropdown-group-item')) {
                const groupId = e.target.dataset.groupId;
                const groupName = e.target.textContent;
                
                if(!isGroupSelected(groupId)) {
                    addSelectedGroup(groupId, groupName);
                }
            }
        });

        function addSelectedGroup(groupId, groupName) {
            const badge = document.createElement('span');
            badge.className = 'badge bg-primary me-1';
            badge.dataset.groupId = groupId;
            badge.innerHTML = `
                ${groupName}
                <button type="button" class="btn-close btn-close-white ms-2"></button>
            `;
            badge.querySelector('button').addEventListener('click', () => {
                badge.remove();
            });
            selectedGroups.appendChild(badge);
        }

        function isGroupSelected(groupId) {
            return Array.from(selectedGroups.children)
                .some(badge => badge.dataset.groupId === groupId);
        }

    // Обработчик фокуса
    searchInput.addEventListener('focus', () => {
        dropdown.classList.add('show'); // Добавляем класс show
    });

    // Обработчик клика вне элемента
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.dropdown-group')) {
            dropdown.classList.remove('show');
        }
    });

    // Обработчик ввода текста
    searchInput.addEventListener('input', function() {
        const filter = this.value.toLowerCase();
        Array.from(dropdown.children).forEach(item => {
            const text = item.textContent.toLowerCase();
            item.style.display = text.includes(filter) ? 'block' : 'none';
        });
    });
    }

    // Обработчик загрузки файлов
    window.uploadFile = function(courseId) {
        const input = document.getElementById(`fileInput${courseId}`);
        const file = input.files[0];
        
        if(file) {
            const formData = new FormData();
            formData.append('file', file);
            formData.append('courseId', courseId);

            fetch('/api/uploadfile', {
                method: 'POST',
                body: formData
            })
            .then(response => {
                if(response.ok) location.reload();
                else alert('Ошибка загрузки файла');
            });
        }
    }

    document.querySelector('.addform button.btn-primary').addEventListener('click', async function() {
    const button = this;
    const spinner = button.querySelector('.loading-spinner');
    
            try {
            button.disabled = true;
            spinner.style.display = 'inline-block';
            // Сбор данных
            const courseData = {
                name: document.getElementById('name').value.trim(),
                description: document.querySelector('.addform textarea').value.trim(),
                groups: Array.from(document.querySelectorAll('.selected-groups .badge'))
                    .map(badge => parseInt(badge.dataset.groupId)),
                access_token: localStorage.getItem('access_token')
            };

            // Валидация
            if (!courseData.name) {
                alert('Введите название курса');
                return;
            }
            
            if (courseData.groups.length === 0) {
                alert('Выберите хотя бы одну группу');
                return;
            }

            // Отправка запроса
            const response = await fetch('/api/createcourse', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(courseData)
            });

            // Обработка ответа
            if (response.ok) {
                const newCourse = await response.json(); // Предполагаем, что сервер возвращает созданный курс
                addCourseToTable(newCourse);
                clearForm();
            } else {
                const error = await response.json();
                alert(`Ошибка: ${error.message || 'Неизвестная ошибка'}`);
            }

        } catch (error) {
            console.error('Ошибка:', error);
            alert('Произошла ошибка при создании курса');
        } finally {
            button.disabled = false;
            spinner.style.display = 'none';
        }
    
    });

    // Функция добавления курса в таблицу
    function addCourseToTable(course) {
        const tbody = document.querySelector('table tbody');
        const newRow = createCourseRow(course);
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = newRow;
        const rowElement = tempDiv.firstElementChild;
        rowElement.classList.add('new-course');
        tbody.insertAdjacentElement('afterbegin', rowElement);
    }
    // Функция создания HTML-строки для курса
    function createCourseRow(course) {
        return `
            <tr data-course-id="${course.id}">
                <td>${course.name}</td>
                <td>
                    <button class="btn btn-info btn-sm" 
                            data-bs-toggle="modal" 
                            data-bs-target="#theoryModal${course.id}">
                        Просмотр
                    </button>
                    ${createTheoryModal(course)}
                </td>
                <td>
                    <button class="btn btn-info btn-sm" 
                            data-bs-toggle="modal" 
                            data-bs-target="#testsModal${course.id}">
                        Просмотр
                    </button>
                    ${createTestsModal(course)}
                </td>
                <td><button class="btn btn-danger btn-sm">Удалить</button></td>
            </tr>
        `;
    }

    // Функции для создания модальных окон
    function createTheoryModal(course) {
        return `
            <div class="modal fade" id="theoryModal${course.id}">
                <!-- Содержимое модального окна как в предыдущей реализации -->
                <!-- Убедитесь, что для новых курсов Files инициализирован как пустой массив -->
            </div>
        `;
    }

    function createTestsModal(course) {
        return `
            <div class="modal fade" id="testsModal${course.id}">
                <!-- Содержимое модального окна как в предыдущей реализации -->
                <!-- Убедитесь, что для новых курсов Tests инициализирован как пустой массив -->
            </div>
        `;
    }

    // Очистка формы
    function clearForm() {
        document.getElementById('name').value = '';
        document.querySelector('.addform textarea').value = '';
        document.querySelectorAll('.selected-groups .badge').forEach(badge => badge.remove());
    }

    document.querySelector('table tbody').addEventListener('click', function(e) {
        if (e.target.classList.contains('btn-danger')) {
            deleteCourse(e.target.closest('tr'));
        }
    });

    async function deleteCourse(row) {
        const courseData = {
            id: row.dataset.courseId,
            token: localStorage.getItem('access_token')
        }
        
        console.log(courseData.id)
        try {
            const response = await fetch(`/api/deletecourse`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(courseData)
            });
            
            if (response.ok) {
                row.remove();
            }
        } catch (error) {
            console.error('Ошибка удаления:', error);
        }
    }
});

function handleRedirect() {
    window.location.href = 'http://localhost:9293/profile';
}