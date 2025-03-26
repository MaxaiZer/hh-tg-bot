# hh-tg-bot

Telegram бот для поиска вакансий на hh.ru с фильтрацией с помощью ИИ (Gemini API).

Зачем ИИ? Чтобы при поиске по ключевому слову "Go" не получить в рекомендации вакансию 1С разработчика, т.к. в описании было "будет плюсом: знание Go". Также можно фильтровать по действительно важным критериям, задав при поиске "хочу вкусняшки в офисе".

## Pitfalls

### Перепубликация вакансии
hh.ru позволяет работодателям переопубликовывать вакансии, что меняет только дату публикации. Поэтому необходимо хранить id уже предложенных вакансий для каждого пользователя.

### Дубликаты вакансий
Некоторые работодатели могут дублировать вакансию под разные города с одинаковым описанием. Но описание может слегка отличаться: может встретиться дополнительный пробел или html тег.
