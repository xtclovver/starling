const errorTranslations: Record<string, string> = {
  'invalid email or password': 'Неверный email или пароль',
  'invalid credentials': 'Неверный email или пароль',
  'user not found': 'Пользователь не найден',
  'user already exists': 'Пользователь с таким email уже существует',
  'username already taken': 'Это имя пользователя уже занято',
  'email already taken': 'Этот email уже занят',
  'email already exists': 'Пользователь с таким email уже существует',
  'username already exists': 'Это имя пользователя уже занято',
  'unauthorized': 'Необходима авторизация',
  'forbidden': 'Доступ запрещён',
  'not found': 'Не найдено',
  'internal server error': 'Ошибка сервера',
  'too many requests': 'Слишком много запросов, попробуйте позже',
  'invalid token': 'Недействительный токен',
  'token expired': 'Срок действия токена истёк',
  'invalid username': 'Некорректное имя пользователя',
  'invalid email': 'Некорректный email',
  'invalid password': 'Некорректный пароль',
  'password too short': 'Пароль слишком короткий',
  'username too short': 'Имя пользователя слишком короткое',
  'username too long': 'Имя пользователя слишком длинное',
  'content too long': 'Текст слишком длинный',
  'post not found': 'Пост не найден',
  'comment not found': 'Комментарий не найден',
  'user is banned': 'Пользователь заблокирован',
  'account is banned': 'Аккаунт заблокирован',
  'you are banned': 'Ваш аккаунт заблокирован',
  'cannot ban yourself': 'Нельзя заблокировать самого себя',
  'cannot admin yourself': 'Нельзя изменить права самого себя',
};

export function translateBackendError(msg: string | undefined | null): string | null {
  if (!msg) return null;
  const lower = msg.toLowerCase();
  if (errorTranslations[lower]) return errorTranslations[lower];
  for (const [key, val] of Object.entries(errorTranslations)) {
    if (lower.includes(key)) return val;
  }
  return msg;
}
