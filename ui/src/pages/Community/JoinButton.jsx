import PropTypes from 'prop-types';
import React from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { communityAdded } from '../../slices/communitiesSlice';
import { loginPromptToggled } from '../../slices/mainSlice';

const JoinButton = ({ className, community, ...rest }) => {
  const loggedIn = useSelector((state) => state.main.user) !== null;
  const dispatch = useDispatch();

  const joined = community ? community.userJoined : false;
  const handleFollow = async () => {
    if (!loggedIn) {
      dispatch(loginPromptToggled());
      return;
    }
    const message = `You will no longer be a moderator of '${community.name}' if you leave the community. Are you sure you want to leave?`;
    if (community.userMod && !confirm(message)) {
      return;
    }
    try {
      const response = await fetch('/api/_joinCommunity', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ communityId: community.id }),
      });
      
      const data = await response.json();
      
      if (!response.ok) 
        if (data.code === 'member-limit-reached' || response.status === 401) {
          alert('This community has reached its maximum member limit of 11.');
          return;
        }
        throw new Error('Failed to join community');
      }

      dispatch(communityAdded(data));
    } catch (error) {
      console.error('Error joining community:', error);
      console.error('Error details:', error.message);
      alert('Failed to join community');
    }
  };

  let cls = joined ? '' : 'button-main';
  if (className) cls += ` ${className}`;

  return (
    <button onClick={handleFollow} className={cls} {...rest}>
      {joined ? 'Joined' : 'Join'}
    </button>
  );
};

JoinButton.propTypes = {
  community: PropTypes.object.isRequired,
  className: PropTypes.string,
};

export default JoinButton;
