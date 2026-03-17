import { useState, useEffect, useCallback } from 'react';
import { getCommentTree } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import CommentItem from './CommentItem';
import CommentForm from './CommentForm';
import Spinner from './Spinner';
import s from '@/styles/profile.module.css';
import type { Comment } from '@/types';

function addReplyToTree(comments: Comment[], parentId: string, reply: Comment): Comment[] {
  return comments.map((c) => {
    if (c.id === parentId) return { ...c, children: [reply, ...(c.children || [])] };
    if (c.children?.length) return { ...c, children: addReplyToTree(c.children, parentId, reply) };
    return c;
  });
}

function removeFromTree(comments: Comment[], commentId: string): Comment[] {
  return comments.map((c) => {
    if (c.id === commentId) {
      if (c.children?.length) return { ...c, content: '[удалено]', user_id: '' };
      return null;
    }
    if (c.children?.length) return { ...c, children: removeFromTree(c.children, commentId) };
    return c;
  }).filter(Boolean) as Comment[];
}

export default function CommentTree({ postId }: { postId: string }) {
  const user = useAuthStore((st) => st.user);
  const [comments, setComments] = useState<Comment[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (c = '') => {
    setLoading(true);
    try {
      const data = await getCommentTree(postId, c);
      if (c) setComments((prev) => [...prev, ...(data.comments || [])]);
      else setComments(data.comments || []);
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch {}
    finally { setLoading(false); }
  }, [postId]);

  useEffect(() => { load(); }, [load]);

  return (
    <div>
      {user && <CommentForm postId={postId} onSubmit={(c) => setComments((prev) => [c, ...prev])} />}
      <div style={{ padding: '0 16px' }}>
        {comments.map((c) => (
          <CommentItem
            key={c.id}
            comment={c}
            postId={postId}
            onNewReply={(pid, reply) => setComments((prev) => addReplyToTree(prev, pid, reply))}
            onDelete={(cid) => setComments((prev) => removeFromTree(prev, cid))}
          />
        ))}
      </div>
      {loading && <Spinner />}
      {hasMore && !loading && (
        <button onClick={() => load(cursor)} className={s.loadMoreBtn}>Загрузить ещё</button>
      )}
      {!loading && comments.length === 0 && <p className={s.empty}>Нет комментариев</p>}
    </div>
  );
}
