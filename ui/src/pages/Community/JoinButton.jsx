import PropTypes from 'prop-types';
import React from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { mfetchjson } from '../../helper';
import { communityAdded } from '../../slices/communitiesSlice';
import { loginPromptToggled, snackAlert } from '../../slices/mainSlice';

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
      const rcomm = await mfetchjson('/api/_joinCommunity', {
        method: 'POST',
        body: JSON.stringify({ communityId: community.id, leave: joined }),
      });
      dispatch(communityAdded(rcomm));
    } catch (error) {
      if (error.name === 'APIError' && error.json && typeof error.json === 'object' && 'message' in error.json) {
        dispatch(snackAlert(error.json.message, 'join_error'));
      } else {
        dispatch(snackAlert('Failed to join community', 'join_error'));
      }
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